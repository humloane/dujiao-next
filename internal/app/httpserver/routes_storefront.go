package httpserver

import (
	"github.com/dujiao-next/internal/app/container"
	"github.com/dujiao-next/internal/app/httpserver/middleware"
	affiliatebootstrap "github.com/dujiao-next/internal/bootstrap/affiliate"
	"github.com/dujiao-next/internal/config"
	affiliatetransport "github.com/dujiao-next/internal/modules/affiliate/transport/http"
	apicredentialtransport "github.com/dujiao-next/internal/modules/apicredential/transport/http"
	auditlogtransport "github.com/dujiao-next/internal/modules/auditlog/transport/http"
	captchatransport "github.com/dujiao-next/internal/modules/captcha/transport/http"
	carttransport "github.com/dujiao-next/internal/modules/cart/transport/http"
	categoryhttp "github.com/dujiao-next/internal/modules/catalog/category/transport/http"
	producthttp "github.com/dujiao-next/internal/modules/catalog/product/transport/http"
	contenttransport "github.com/dujiao-next/internal/modules/content/transport/http"
	giftcardtransport "github.com/dujiao-next/internal/modules/giftcard/transport/http"
	userauthtransport "github.com/dujiao-next/internal/modules/identity/userauth/transport/http"
	memberleveltransport "github.com/dujiao-next/internal/modules/memberlevel/transport/http"
	ordertransport "github.com/dujiao-next/internal/modules/order/transport/http"
	paymenttransport "github.com/dujiao-next/internal/modules/payment/transport/http"
	paymentcallbacktransport "github.com/dujiao-next/internal/modules/payment/transport/http/callback"
	resellertransport "github.com/dujiao-next/internal/modules/reseller/transport/http/user"
	publicconfigtransport "github.com/dujiao-next/internal/modules/settings/transport/http/public"
	wallettransport "github.com/dujiao-next/internal/modules/wallet/transport/http"

	"context"
	"encoding/json"
	"fmt"
	"github.com/dujiao-next/internal/constants"
	"github.com/dujiao-next/internal/htmltext"
	categorydomain "github.com/dujiao-next/internal/modules/catalog/category/domain"
	productcontract "github.com/dujiao-next/internal/modules/catalog/product/contract"
	productdomain "github.com/dujiao-next/internal/modules/catalog/product/domain"
	contentapp "github.com/dujiao-next/internal/modules/content/application"
	contentdomain "github.com/dujiao-next/internal/modules/content/domain"
	"github.com/dujiao-next/internal/shared/jsonmap"
	"github.com/dujiao-next/internal/web"
	"github.com/dujiao-next/internal/web/lpcontent"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"html"
	"net/url"
	"path"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

func registerStorefrontRoutes(
	apiV1 *gin.RouterGroup,
	cfg *config.Config,
	c *container.Container,
	publicContentHandler *contenttransport.PublicHandler,
	publicCatalogHandler *producthttp.PublicHandler,
	publicCategoryHandler *categoryhttp.PublicHandler,
	userResellerHandler *resellertransport.UserHandler,
	userResellerProductSettingHandler *resellertransport.UserProductSettingHandler,
	userResellerFinanceHandler *resellertransport.UserFinanceHandler,
	userResellerOrderHandler *resellertransport.UserOrderHandler,
	userApiCredentialHandler *apicredentialtransport.UserHandler,
	userAuditLogHandler *auditlogtransport.UserHandler,
	userGiftCardHandler *giftcardtransport.UserHandler,
	publicMemberLevelHandler *memberleveltransport.PublicHandler,
	userProfileHandler *userauthtransport.UserProfileHandler,
	userEmailHandler *userauthtransport.UserEmailHandler,
	userPasswordHandler *userauthtransport.UserPasswordHandler,
	userVerifyHandler *userauthtransport.UserVerifyHandler,
	userTelegramOIDCHandler *userauthtransport.UserTelegramOIDCHandler,
	userTelegramHandler *userauthtransport.UserTelegramHandler,
	userGoogleHandler *userauthtransport.UserGoogleHandler,
	userLoginHandler *userauthtransport.UserLoginHandler,
	user2FAHandler *userauthtransport.User2FAHandler,
	userInviteHandler *userauthtransport.UserInviteHandler,
	publicConfigHandler *publicconfigtransport.Handler,
	userCartHandler *carttransport.UserHandler,
	userOrderHandler *ordertransport.UserHandler,
	guestOrderHandler *ordertransport.GuestHandler,
	orderPreviewHandler *ordertransport.PreviewHandler,
	orderCreateHandler *ordertransport.CreateHandler,
	paymentLatestHandler *paymenttransport.LatestHandler,
	paymentWriteHandler *paymenttransport.WriteHandler,
	userWalletHandler *wallettransport.UserHandler,
	redisClient *redis.Client,
	loginRule middleware.RateLimitRule,
	guestReadRule middleware.RateLimitRule,
	guestWriteRule middleware.RateLimitRule,
	giftCardRedeemRule middleware.RateLimitRule,
) {
	storefront := apiV1.Group("")
	storefront.Use(middleware.ResellerTenantMiddleware(c.ResellerDomainResolver))
	affiliateHandler := affiliatebootstrap.NewStorefrontHandler(c)

	// 公开接口
	public := storefront.Group("/public")
	{
		publicconfigtransport.RegisterPublicRoutes(public, publicConfigHandler)
		producthttp.RegisterPublicRoutes(public, publicCatalogHandler)
		categoryhttp.RegisterPublicRoutes(public, publicCategoryHandler)
		contenttransport.RegisterPublicRoutes(public, publicContentHandler)
		captchatransport.RegisterPublicRoutes(public, captchatransport.NewPublicHandler(c.CaptchaService))
		affiliatetransport.RegisterPublicRoutes(public, affiliateHandler)
		memberleveltransport.RegisterPublicRoutes(public, publicMemberLevelHandler)
	}

	// 游客接口
	guest := storefront.Group("/guest")
	guestRead := guest.Group("")
	guestRead.Use(middleware.RateLimitMiddleware(redisClient, guestReadRule, middleware.KeyByIP))
	{
		ordertransport.RegisterGuestPreviewRoute(guestRead, orderPreviewHandler)
		ordertransport.RegisterGuestReadRoutes(guestRead, guestOrderHandler)
		paymenttransport.RegisterGuestLatestRoute(guestRead, paymentLatestHandler)
	}
	guestWrite := guest.Group("")
	guestWrite.Use(middleware.RateLimitMiddleware(redisClient, guestWriteRule, middleware.KeyByIP))
	{
		ordertransport.RegisterGuestCreateRoute(guestWrite, orderCreateHandler)
		ordertransport.RegisterGuestCreateAndPayRoute(guestWrite, orderCreateHandler)
		paymenttransport.RegisterGuestWriteRoutes(guestWrite, paymentWriteHandler)
	}

	// 注册 / 忘记密码 / 发送验证码：按 IP 计数，共享出口 IP（NAT）下一次完整流程就要 2~4 次请求，
	// 不能照搬登录的 5 次/封 15 分钟，否则同网用户会被成批误封。
	authRule := middleware.RateLimitRule{
		Prefix:        loginRule.Prefix + ":auth",
		WindowSeconds: 300,
		MaxRequests:   20,
		BlockSeconds:  300,
		MessageKey:    "error.rate_limited",
	}
	authRateLimit := middleware.RateLimitMiddleware(redisClient, authRule, middleware.KeyByIP)

	// 用户认证接口
	auth := storefront.Group("/auth")
	{
		userauthtransport.RegisterUserVerifyAuthRoutes(auth, userVerifyHandler, authRateLimit)
		userauthtransport.RegisterUserRegisterAuthRoutes(auth, userLoginHandler, authRateLimit)
		userauthtransport.RegisterUserLoginAuthRoutes(auth, userLoginHandler, middleware.RateLimitMiddleware(redisClient, loginRule, middleware.KeyByIPAndJSONField("email")))
		userauthtransport.RegisterUser2FAAuthRoutes(auth, user2FAHandler, middleware.RateLimitMiddleware(redisClient, loginRule, middleware.KeyByIP))
		userauthtransport.RegisterUserTelegramAuthRoutes(auth, userTelegramHandler, middleware.RateLimitMiddleware(redisClient, loginRule, middleware.KeyByIP))
		userauthtransport.RegisterUserTelegramOIDCAuthRoutes(auth, userTelegramOIDCHandler, middleware.RateLimitMiddleware(redisClient, loginRule, middleware.KeyByIP))
		userauthtransport.RegisterUserGoogleAuthRoutes(auth, userGoogleHandler, middleware.RateLimitMiddleware(redisClient, loginRule, middleware.KeyByIP))
		userauthtransport.RegisterUserInviteAuthRoutes(auth, userInviteHandler, middleware.RateLimitMiddleware(redisClient, loginRule, middleware.KeyByIP))
		userauthtransport.RegisterUserPasswordAuthRoutes(auth, userPasswordHandler, authRateLimit)
	}

	// 用户接口（需鉴权）
	user := storefront.Group("")
	user.Use(middleware.UserJWTAuthMiddleware(cfg.UserJWT.SecretKey, c.UserStore))
	{
		userauthtransport.RegisterUserProfileRoutes(user, userProfileHandler)
		auditlogtransport.RegisterUserRoutes(user, userAuditLogHandler)
		userauthtransport.RegisterUserPasswordRoutes(user, userPasswordHandler)
		userauthtransport.RegisterUserTelegramRoutes(user, userTelegramHandler)
		userauthtransport.RegisterUserTelegramOIDCRoutes(user, userTelegramOIDCHandler)
		userauthtransport.RegisterUserGoogleRoutes(
			user,
			userGoogleHandler,
			middleware.RateLimitMiddleware(redisClient, loginRule, middleware.KeyByUserIDAndIP),
		)
		userauthtransport.RegisterUserEmailRoutes(user, userEmailHandler)
		userauthtransport.RegisterUser2FARoutes(user, user2FAHandler)
		carttransport.RegisterUserRoutes(user, userCartHandler)
		ordertransport.RegisterUserCreateRoute(user, orderCreateHandler)
		ordertransport.RegisterUserCreateAndPayRoute(user, orderCreateHandler)
		ordertransport.RegisterUserPreviewRoute(user, orderPreviewHandler)
		ordertransport.RegisterUserPaymentChannelsRoute(user, userOrderHandler)
		ordertransport.RegisterUserReadRoutes(user, userOrderHandler)
		ordertransport.RegisterUserCancelRoute(user, userOrderHandler)
		paymenttransport.RegisterUserWriteRoutes(user, paymentWriteHandler)
		paymenttransport.RegisterUserLatestRoute(user, paymentLatestHandler)
		wallettransport.RegisterUserRoutes(user, userWalletHandler)
		giftCardRedeem := user.Group("", middleware.RateLimitMiddleware(redisClient, giftCardRedeemRule, middleware.KeyByUserIDAndIP))
		giftcardtransport.RegisterUserRoutes(giftCardRedeem, userGiftCardHandler)
		affiliatetransport.RegisterUserRoutes(user, affiliateHandler)

		resellerConsole := user.Group("/reseller")
		resellerConsole.Use(middleware.RequireMainTenantForResellerConsole())
		{
			resellertransport.RegisterUserConsoleRoutes(resellerConsole, userResellerHandler)
			resellertransport.RegisterUserProductSettingRoutes(resellerConsole, userResellerProductSettingHandler)
			resellertransport.RegisterUserFinanceRoutes(resellerConsole, userResellerFinanceHandler)
			resellertransport.RegisterUserOrderRoutes(resellerConsole, userResellerOrderHandler)
		}

		// API 对接权限（用户中心）
		apicredentialtransport.RegisterUserRoutes(user, userApiCredentialHandler)
	}
}

func registerPaymentCallbackRoutes(apiV1 *gin.RouterGroup, callbackHandler *paymentcallbacktransport.Handler, webhookHandler *paymenttransport.WebhookHandler, epayRedirectHandler *paymenttransport.EpayRedirectHandler, redisClient *redis.Client, callbackRule middleware.RateLimitRule) {
	callbacks := apiV1.Group("", middleware.RateLimitMiddleware(redisClient, callbackRule, middleware.KeyByIP))
	paymentcallbacktransport.RegisterRoutes(callbacks, callbackHandler)
	paymenttransport.RegisterWebhookRoutes(callbacks, webhookHandler)
	paymenttransport.RegisterEpayRedirectRoute(callbacks, epayRedirectHandler)
}

const (
	seoDescriptionMaxRunes = 160
	seoCacheTTL            = 60 * time.Second
	seoDefaultLocale       = "zh-CN"
	// defaultOGImagePath 是内置的 1200x630 分享图（frontend/user/public/og-default.png）。
	defaultOGImagePath = "/og-default.png"
)

type seoCacheEntry struct {
	meta    *web.PageMeta
	expires time.Time
}

type seoResolver struct {
	container *container.Container

	mu    sync.Mutex
	cache map[string]seoCacheEntry
}

type seoSite struct {
	Name        string
	URL         string
	Description map[string]interface{}
	SeoTitle    map[string]interface{}
	SeoDesc     map[string]interface{}
	SeoKeywords map[string]interface{}
	Image       string
	// DefaultOGImage 用于 og:image 回退；与站点 Logo 分开，避免分享图覆盖品牌 Logo。
	DefaultOGImage string
	Currency       string
	Telegram       string
	// About / Legal 来自站点设置，用于静态页正文；未配置时正文退化为站点信息。
	About map[string]interface{}
	Legal map[string]interface{}
}

// newSEOMetaResolver 把站点配置与目录服务组合成 web.MetaProvider。
func newSEOMetaResolver(c *container.Container) web.MetaProvider {
	if c == nil {
		return nil
	}
	resolver := &seoResolver{container: c, cache: make(map[string]seoCacheEntry)}
	return resolver.resolve
}

func (r *seoResolver) resolve(req web.MetaRequest) *web.PageMeta {
	locale := resolveSEOLocale(req)
	cacheKey := req.Host + "|" + locale + "|" + req.Path
	if cached := r.readCache(cacheKey); cached != nil {
		return cached
	}

	site := r.loadSite()
	if strings.TrimSpace(site.URL) == "" {
		scheme := strings.TrimSpace(req.Scheme)
		if scheme == "" {
			scheme = "https"
		}
		site.URL = scheme + "://" + strings.TrimSpace(req.Host)
	}
	site.URL = strings.TrimRight(site.URL, "/")
	if site.Name == "" {
		site.Name = "Dujiao-Next"
	}

	meta := r.buildMeta(req, locale, site)
	r.writeCache(cacheKey, meta)
	return meta
}

func (r *seoResolver) readCache(key string) *web.PageMeta {
	r.mu.Lock()
	defer r.mu.Unlock()
	entry, ok := r.cache[key]
	if !ok || time.Now().After(entry.expires) {
		delete(r.cache, key)
		return nil
	}
	return entry.meta
}

func (r *seoResolver) writeCache(key string, meta *web.PageMeta) {
	if meta == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	// 防止长期运行后 map 无限增长：只保留最近 512 个页面。
	if len(r.cache) > 512 {
		r.cache = make(map[string]seoCacheEntry)
	}
	r.cache[key] = seoCacheEntry{meta: meta, expires: time.Now().Add(seoCacheTTL)}
}

func (r *seoResolver) loadSite() seoSite {
	site := seoSite{
		Name:        "Dujiao-Next",
		Description: map[string]interface{}{},
		SeoTitle:    map[string]interface{}{},
		SeoDesc:     map[string]interface{}{},
		SeoKeywords: map[string]interface{}{},
		Currency:    constants.SiteCurrencyDefault,
	}
	if r == nil || r.container == nil || r.container.SettingService == nil {
		return site
	}
	raw, err := r.container.SettingService.GetByKey(constants.SettingKeySiteConfig)
	if err != nil || raw == nil {
		return site
	}
	brand := asMap(raw["brand"])
	if name := asString(brand["site_name"]); name != "" {
		site.Name = name
	}
	site.URL = asString(brand["site_url"])
	site.Description = asMap(brand["site_description"])
	site.Image = asString(brand["site_logo"])
	if site.Image == "" {
		site.Image = asString(brand["site_icon"])
	}

	seo := asMap(raw["seo"])
	site.SeoTitle = asMap(seo["title"])
	site.SeoDesc = asMap(seo["description"])
	site.SeoKeywords = asMap(seo["keywords"])
	if image := asString(seo["default_og_image"]); image != "" {
		site.DefaultOGImage = image
	}
	if currency := asString(raw[constants.SettingFieldSiteCurrency]); currency != "" {
		site.Currency = currency
	}
	site.Telegram = asString(asMap(raw["contact"])["telegram"])
	site.About = asMap(raw["about"])
	site.Legal = asMap(raw["legal"])
	return site
}

func (r *seoResolver) buildMeta(req web.MetaRequest, locale string, site seoSite) *web.PageMeta {
	cleanPath := cleanSEOPath(req.Path)
	base := strings.TrimRight(site.URL, "/")
	canonical := base + cleanPath
	meta := &web.PageMeta{
		Title:       localized(site.SeoTitle, locale),
		Description: localized(site.SeoDesc, locale),
		Keywords:    localized(site.SeoKeywords, locale),
		Canonical:   canonical,
		Image:       absoluteAssetURL(base, site.DefaultOGImage),
		Type:        "website",
		Locale:      locale,
		Path:        cleanPath,
	}
	if meta.Title == "" {
		meta.Title = site.Name
	}
	if meta.Description == "" {
		meta.Description = localized(site.Description, locale)
	}
	if strings.TrimSpace(meta.Image) == "" && base != "" {
		// 站点没配默认分享图时回落到内置 1200x630 图，避免 og:image 为空。
		meta.Image = base + defaultOGImagePath
	}

	// 标题后缀优先用当前语言的站点 SEO 标题，避免英文页面出现中文品牌名。
	siteName := seoSiteName(site, locale)

	ctx := req.Context
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	switch {
	case cleanPath == "/" || cleanPath == "":
		meta.Title = firstNonEmpty(localized(site.SeoTitle, locale), site.Name)
		meta.Description = firstNonEmpty(localized(site.SeoDesc, locale), localized(site.Description, locale), meta.Description)
		meta.JSONLD = marshalJSONLD(homeJSONLD(base, site, locale))
		meta.BodyHTML = r.homeBodyHTML(ctx, locale, site.Name, meta.Description, site)
	case cleanPath == "/products":
		meta.Title = joinTitle(staticText("products", locale), siteName)
		meta.Description = firstNonEmpty(localized(site.SeoDesc, locale), staticText("productsDesc", locale), meta.Description)
		meta.JSONLD = marshalJSONLD(collectionJSONLD(base, cleanPath, meta.Title, meta.Description, site, locale))
		meta.BodyHTML = r.productsBodyHTML(locale, staticText("products", locale), meta.Description, site)
	case strings.HasPrefix(cleanPath, "/categories/"):
		slug := strings.TrimPrefix(cleanPath, "/categories/")
		if category := r.findCategory(slug); category != nil {
			name := localized(asMap(category.NameJSON), locale)
			seo := asMap(category.SeoMetaJSON)
			meta.Title = joinTitle(firstNonEmpty(asStringFromMap(seo["title"], locale), name), siteName)
			meta.Description = truncateRunes(firstNonEmpty(
				asStringFromMap(seo["description"], locale),
				localized(asMap(category.DescriptionJSON), locale),
				meta.Description,
				staticText("productsDesc", locale),
			), seoDescriptionMaxRunes)
			meta.Keywords = asStringFromMap(seo["keywords"], locale)
			meta.JSONLD = marshalJSONLD(collectionJSONLD(base, cleanPath, meta.Title, meta.Description, site, locale))
			meta.BodyHTML = r.categoryBodyHTML(locale, site, category, meta.Description)
		}
	case strings.HasPrefix(cleanPath, "/products/"):
		slug := strings.TrimPrefix(cleanPath, "/products/")
		if product := r.findProduct(slug); product != nil {
			r.applyProductMeta(ctx, meta, base, cleanPath, locale, site, product)
		}
	case strings.HasPrefix(cleanPath, "/blog/"):
		slug := strings.TrimPrefix(cleanPath, "/blog/")
		if post := r.findPost(ctx, slug); post != nil {
			r.applyPostMeta(meta, base, cleanPath, locale, site, post)
		}
	case cleanPath == "/blog" || cleanPath == "/notice":
		key := strings.TrimPrefix(cleanPath, "/")
		meta.Title = joinTitle(staticText(key, locale), siteName)
		meta.Description = firstNonEmpty(staticText(key+"Desc", locale), meta.Description)
		postType := constants.PostTypeBlog
		if cleanPath == "/notice" {
			postType = constants.PostTypeNotice
		}
		posts := r.seoPosts(ctx, postType, blogBodyPostLimit)
		// 列表页始终输出一份 CollectionPage + BreadcrumbList；即使暂无内容，
		// 也比空 JSON-LD 更利于搜索引擎理解页面结构。
		meta.JSONLD = marshalJSONLD(collectionJSONLD(base, cleanPath, meta.Title, meta.Description, site, locale))
		if cleanPath == "/blog" {
			meta.BodyHTML = r.blogListBodyHTML(locale, site, meta.Title, meta.Description, posts)
		} else {
			meta.BodyHTML = r.noticeListBodyHTML(locale, site, meta.Title, meta.Description, posts)
		}
	case cleanPath == "/about" || cleanPath == "/terms" || cleanPath == "/privacy":
		key := strings.TrimPrefix(cleanPath, "/")
		meta.Title = joinTitle(staticText(key, locale), siteName)
		meta.Description = firstNonEmpty(staticText(key+"Desc", locale), meta.Description)
		meta.JSONLD = marshalJSONLD(staticPageJSONLD(base, cleanPath, meta.Title, meta.Description, site, locale))
		meta.BodyHTML = r.staticPageBodyHTML(locale, key, staticText(key, locale), meta.Description, site)
	case strings.HasPrefix(cleanPath, "/lp/"):
		slug := strings.TrimPrefix(cleanPath, "/lp/")
		page, loc, ok := lpcontent.Lookup(locale, slug)
		if !ok {
			meta.Robots = "noindex, nofollow"
			break
		}
		// 落地页是独立销售页：用 WebPage 而不是 CollectionPage，
		// 关键词也来自落地页自己的文案，不复用站点 keywords。
		meta.Title = joinTitle(page.Title, siteName)
		meta.Description = truncateRunes(firstNonEmpty(page.Subtitle, meta.Description), seoDescriptionMaxRunes)
		meta.Keywords = strings.TrimSpace(page.Keywords)
		meta.JSONLD = marshalJSONLD(landingJSONLD(base, cleanPath, page, site, locale))
		meta.BodyHTML = landingBodyHTML(page, loc)
	case isPrivateSEOPath(cleanPath):
		meta.Robots = "noindex, nofollow"
	default:
		meta.Robots = "noindex, nofollow"
	}

	if strings.TrimSpace(meta.Title) == "" {
		meta.Title = site.Name
	}
	if meta.Robots == "" {
		meta.Alternates = buildAlternates(base, cleanPath)
	}
	return meta
}

func (r *seoResolver) findCategory(slug string) *categorydomain.Category {
	if r == nil || r.container == nil || r.container.CategoryService == nil {
		return nil
	}
	slug = strings.TrimSpace(slug)
	if slug == "" {
		return nil
	}
	categories, err := r.container.CategoryService.ListActive()
	if err != nil {
		return nil
	}
	for i := range categories {
		if categories[i].Slug == slug {
			return &categories[i]
		}
	}
	return nil
}

func (r *seoResolver) findProduct(slug string) *productdomain.Product {
	if r == nil || r.container == nil || r.container.ProductReadService == nil {
		return nil
	}
	slug = strings.TrimSpace(slug)
	if slug == "" {
		return nil
	}
	product, err := r.container.ProductReadService.GetPublicBySlug(slug)
	if err != nil {
		return nil
	}
	return product
}

func (r *seoResolver) findPost(ctx context.Context, slug string) *contentdomain.Post {
	if r == nil || r.container == nil || r.container.ContentPostService == nil {
		return nil
	}
	slug = strings.TrimSpace(slug)
	if slug == "" {
		return nil
	}
	post, err := r.container.ContentPostService.GetPublicBySlug(ctx, slug)
	if err != nil {
		return nil
	}
	return post
}

func (r *seoResolver) applyProductMeta(ctx context.Context, meta *web.PageMeta, base, cleanPath, locale string, site seoSite, product *productdomain.Product) {
	seo := asMap(product.SeoMetaJSON)
	title := localized(asMap(product.TitleJSON), locale)
	if title == "" {
		title = product.Slug
	}
	meta.Title = joinTitle(firstNonEmpty(asStringFromMap(seo["title"], locale), title), seoSiteName(site, locale))
	meta.Description = truncateRunes(firstNonEmpty(
		asStringFromMap(seo["description"], locale),
		stripHTML(localized(asMap(product.DescriptionJSON), locale)),
		localized(site.SeoDesc, locale),
	), seoDescriptionMaxRunes)
	meta.Keywords = asStringFromMap(seo["keywords"], locale)
	meta.Type = "product"

	images := make([]string, 0, len(product.Images))
	for _, image := range product.Images {
		if absolute := absoluteAssetURL(base, image); absolute != "" {
			images = append(images, absolute)
		}
	}
	if len(images) > 0 {
		meta.Image = images[0]
	}

	canonical := base + cleanPath
	productJSONLD := map[string]interface{}{
		"@type":       "Product",
		"name":        title,
		"description": meta.Description,
		"url":         canonical,
		"sku":         product.Slug,
		"brand":       map[string]interface{}{"@type": "Brand", "name": site.Name},
		"offers": map[string]interface{}{
			"@type":         "Offer",
			"url":           canonical,
			"price":         product.PriceAmount.String(),
			"priceCurrency": site.Currency,
			"availability":  productAvailability(product),
		},
	}
	if len(images) > 0 {
		productJSONLD["image"] = images
	}
	// Product + BreadcrumbList 用 @graph 输出，Offer 字段保持原样。
	meta.JSONLD = marshalJSONLD(map[string]interface{}{
		"@context": "https://schema.org",
		"@graph": []interface{}{
			productJSONLD,
			map[string]interface{}{
				"@type": "BreadcrumbList",
				"itemListElement": []interface{}{
					map[string]interface{}{"@type": "ListItem", "position": 1, "name": site.Name, "item": base + "/"},
					map[string]interface{}{"@type": "ListItem", "position": 2, "name": staticText("products", locale), "item": base + "/products"},
					map[string]interface{}{"@type": "ListItem", "position": 3, "name": title, "item": canonical},
				},
			},
		},
	})
	meta.BodyHTML = r.productBodyHTML(ctx, locale, site, product)
}

func (r *seoResolver) applyPostMeta(meta *web.PageMeta, base, cleanPath, locale string, site seoSite, post *contentdomain.Post) {
	seo := asMap(post.SeoMetaJSON)
	title := localized(asMap(post.TitleJSON), locale)
	if title == "" {
		title = post.Slug
	}
	meta.Title = joinTitle(firstNonEmpty(asStringFromMap(seo["title"], locale), title), seoSiteName(site, locale))
	meta.Description = truncateRunes(firstNonEmpty(
		asStringFromMap(seo["description"], locale),
		stripHTML(localized(asMap(post.SummaryJSON), locale)),
		stripHTML(localized(asMap(post.ContentJSON), locale)),
		localized(site.SeoDesc, locale),
	), seoDescriptionMaxRunes)
	meta.Keywords = asStringFromMap(seo["keywords"], locale)
	meta.Type = "article"
	if image := absoluteAssetURL(base, post.Thumbnail); image != "" {
		meta.Image = image
	}

	canonical := base + cleanPath
	published := post.CreatedAt
	if post.PublishedAt != nil {
		published = *post.PublishedAt
	}
	jsonLD := map[string]interface{}{
		"@context":         "https://schema.org",
		"@type":            "Article",
		"headline":         title,
		"description":      meta.Description,
		"url":              canonical,
		"datePublished":    published.UTC().Format(time.RFC3339),
		"dateModified":     post.CreatedAt.UTC().Format(time.RFC3339),
		"author":           map[string]interface{}{"@type": "Organization", "name": site.Name},
		"publisher":        map[string]interface{}{"@type": "Organization", "name": site.Name},
		"mainEntityOfPage": canonical,
	}
	if meta.Image != "" {
		jsonLD["image"] = []string{meta.Image}
	}
	// Article + BreadcrumbList 一起输出，和商品/分类页保持一致的 @graph 结构。
	graph := map[string]interface{}{
		"@context": "https://schema.org",
		"@graph": []interface{}{
			jsonLD,
			map[string]interface{}{
				"@type": "BreadcrumbList",
				"itemListElement": []interface{}{
					map[string]interface{}{"@type": "ListItem", "position": 1, "name": site.Name, "item": base + "/"},
					map[string]interface{}{"@type": "ListItem", "position": 2, "name": title, "item": canonical},
				},
			},
		},
	}
	meta.JSONLD = marshalJSONLD(graph)
	meta.BodyHTML = r.postBodyHTML(locale, site, post)
}

func homeJSONLD(base string, site seoSite, locale string) map[string]interface{} {
	org := map[string]interface{}{
		"@type": "Organization",
		"name":  site.Name,
		"url":   base,
	}
	if site.Image != "" {
		org["logo"] = absoluteAssetURL(base, site.Image)
	}
	if site.Telegram != "" {
		org["sameAs"] = []string{site.Telegram}
	}
	website := map[string]interface{}{
		"@type":      "WebSite",
		"name":       site.Name,
		"url":        base,
		"inLanguage": locale,
		"potentialAction": map[string]interface{}{
			"@type": "SearchAction",
			"target": map[string]interface{}{
				"@type":       "EntryPoint",
				"urlTemplate": base + "/products?search={search_term_string}",
			},
			"query-input": "required name=search_term_string",
		},
	}
	return map[string]interface{}{
		"@context": "https://schema.org",
		"@graph":   []interface{}{org, website},
	}
}

func collectionJSONLD(base, cleanPath, title, description string, site seoSite, locale string) map[string]interface{} {
	items := []interface{}{
		map[string]interface{}{"@type": "ListItem", "position": 1, "name": site.Name, "item": base + "/"},
	}
	switch {
	case strings.HasPrefix(cleanPath, "/categories/"):
		items = append(items, map[string]interface{}{
			"@type": "ListItem", "position": 2, "name": title, "item": base + cleanPath,
		})
	case cleanPath != "/products":
		name := strings.TrimPrefix(cleanPath, "/")
		items = append(items, map[string]interface{}{
			"@type": "ListItem", "position": 2, "name": name, "item": base + cleanPath,
		})
	}
	return map[string]interface{}{
		"@context": "https://schema.org",
		"@graph": []interface{}{
			map[string]interface{}{
				"@type":       "CollectionPage",
				"name":        title,
				"description": description,
				"url":         base + cleanPath,
				"inLanguage":  locale,
			},
			map[string]interface{}{
				"@type":           "BreadcrumbList",
				"itemListElement": items,
			},
		},
	}
}

func productAvailability(product *productdomain.Product) string {
	if product == nil {
		return "https://schema.org/OutOfStock"
	}
	if product.ManualStockTotal == -1 || product.ManualStockTotal > 0 {
		return "https://schema.org/InStock"
	}
	for _, sku := range product.SKUs {
		if sku.IsActive && (sku.ManualStockTotal == -1 || sku.ManualStockTotal > 0) {
			return "https://schema.org/InStock"
		}
	}
	if product.FulfillmentType != constants.FulfillmentTypeManual {
		return "https://schema.org/InStock"
	}
	return "https://schema.org/OutOfStock"
}

func resolveSEOLocale(req web.MetaRequest) string {
	if req.RawQuery != "" {
		if values, err := url.ParseQuery(req.RawQuery); err == nil {
			if lang := strings.TrimSpace(values.Get("lang")); lang != "" {
				if normalized := normalizeLocale(lang); normalized != "" {
					return normalized
				}
			}
		}
	}
	accept := strings.ToLower(req.AcceptLanguage)
	switch {
	case strings.Contains(accept, "zh-tw"), strings.Contains(accept, "zh-hk"), strings.Contains(accept, "zh-hant"):
		return "zh-TW"
	case strings.Contains(accept, "zh"):
		return "zh-CN"
	case strings.Contains(accept, "en"):
		return "en-US"
	default:
		return seoDefaultLocale
	}
}

func normalizeLocale(raw string) string {
	value := strings.ToLower(strings.TrimSpace(raw))
	switch {
	case value == "zh-cn" || value == "zh":
		return "zh-CN"
	case value == "zh-tw" || value == "zh-hk" || value == "zh-hant":
		return "zh-TW"
	case value == "en" || value == "en-us":
		return "en-US"
	default:
		return ""
	}
}

func cleanSEOPath(raw string) string {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "/"
	}
	if decoded, err := url.PathUnescape(value); err == nil {
		value = decoded
	}
	value = path.Clean(value)
	if value == "." {
		return "/"
	}
	if !strings.HasPrefix(value, "/") {
		value = "/" + value
	}
	return value
}

func isPrivateSEOPath(cleanPath string) bool {
	privatePrefixes := []string{
		"/me", "/cart", "/checkout", "/pay", "/orders", "/recharge-orders",
		"/guest", "/auth", "/reseller",
	}
	for _, prefix := range privatePrefixes {
		if cleanPath == prefix || strings.HasPrefix(cleanPath, prefix+"/") {
			return true
		}
	}
	return false
}

func buildAlternates(base, cleanPath string) map[string]string {
	return map[string]string{
		"zh-CN":     base + cleanPath + "?lang=zh-CN",
		"zh-TW":     base + cleanPath + "?lang=zh-TW",
		"en-US":     base + cleanPath + "?lang=en-US",
		"x-default": base + cleanPath,
	}
}

func joinTitle(page, site string) string {
	page = strings.TrimSpace(page)
	site = strings.TrimSpace(site)
	if page == "" {
		return site
	}
	if site == "" || titleContainsSite(page, site) {
		return page
	}
	suffix := firstNonEmpty(titleBrandCore(site), site)
	return page + " - " + suffix
}

// titleBrandCore 取站点名第一段作为品牌核心，例如
// "霓途 Neaixt｜全球静态住宅 IP" -> "霓途 Neaixt"。
func titleBrandCore(value string) string {
	value = strings.TrimSpace(value)
	for _, separator := range []string{"｜", "|", "·", "•", "—", "–", ":", "："} {
		if index := strings.Index(value, separator); index > 0 {
			value = strings.TrimSpace(value[:index])
		}
	}
	return value
}

// titleContainsSite 判断页面标题是否已经包含站点名或品牌核心（忽略大小写/空白差异）。
func titleContainsSite(page, site string) bool {
	pageKey := normalizeTitleForMatch(page)
	siteKey := normalizeTitleForMatch(site)
	if pageKey == "" {
		return false
	}
	if siteKey == "" {
		return true
	}
	if strings.Contains(pageKey, siteKey) {
		return true
	}
	coreKey := normalizeTitleForMatch(titleBrandCore(site))
	return len([]rune(coreKey)) >= 2 && strings.Contains(pageKey, coreKey)
}

func normalizeTitleForMatch(value string) string {
	return strings.ToLower(strings.Join(strings.Fields(value), " "))
}

// seoSiteName 返回标题用的站点名：优先当前语言的 seo.title，回落到品牌名；
// 返回前截掉副标题，避免把整段营销文案追加到每个页面标题后面。
func seoSiteName(site seoSite, locale string) string {
	name := firstNonEmpty(localized(site.SeoTitle, locale), site.Name)
	return firstNonEmpty(titleBrandCore(name), name)
}

func truncateRunes(value string, max int) string {
	value = strings.TrimSpace(value)
	if max <= 0 || utf8.RuneCountInString(value) <= max {
		return value
	}
	runes := []rune(value)
	return strings.TrimSpace(string(runes[:max])) + "…"
}

func stripHTML(raw string) string {
	return strings.TrimSpace(htmltext.StripToPlainText(raw))
}

func asMap(value interface{}) map[string]interface{} {
	switch typed := value.(type) {
	case map[string]interface{}:
		return typed
	case jsonmap.JSON:
		return map[string]interface{}(typed)
	default:
		return map[string]interface{}{}
	}
}

func asString(value interface{}) string {
	text, _ := value.(string)
	return strings.TrimSpace(text)
}

// asStringFromMap 读取 SEO 子字段：既支持多语言 map，也支持纯字符串。
func asStringFromMap(value interface{}, locale string) string {
	if text := asString(value); text != "" {
		return text
	}
	return localized(asMap(value), locale)
}

func localized(value map[string]interface{}, locale string) string {
	if len(value) == 0 {
		return ""
	}
	if text := asString(value[locale]); text != "" {
		return text
	}
	for _, fallback := range []string{seoDefaultLocale, "en-US"} {
		if text := asString(value[fallback]); text != "" {
			return text
		}
	}
	for _, item := range value {
		if text := asString(item); text != "" {
			return text
		}
	}
	return ""
}

func absoluteAssetURL(base, raw string) string {
	value := strings.TrimSpace(raw)
	if value == "" {
		return ""
	}
	if strings.HasPrefix(value, "http://") || strings.HasPrefix(value, "https://") {
		return value
	}
	if !strings.HasPrefix(value, "/") {
		value = "/" + value
	}
	return strings.TrimRight(base, "/") + value
}

func marshalJSONLD(value interface{}) string {
	if value == nil {
		return ""
	}
	body, err := json.Marshal(value)
	if err != nil {
		return ""
	}
	return string(body)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

// staticText 返回服务端已知的静态页/落地页 SEO 文案，避免原始 HTML 只有空 title。
func staticText(key, locale string) string {
	texts := map[string][3]string{
		"products":                 {"商品中心", "商品中心", "Products"},
		"productsDesc":             {"浏览全部静态住宅 IP、IEPL 专线与跨境网络方案。", "瀏覽全部靜態住宅 IP、IEPL 專線與跨境網路方案。", "Browse static residential IPs, IEPL lines and cross-border networking plans."},
		"blog":                     {"博客", "部落格", "Blog"},
		"blogDesc":                 {"阅读跨境网络、静态住宅 IP 与直播运营的实用指南，内容涵盖线路选择、节点维护、设备配置、支付与交付说明。文章按发布时间更新，帮助跨境电商、直播团队和多店铺运营者了解可用方案、常见问题与最佳实践；如需针对具体地区或场景的建议，可通过页面上的联系方式咨询。", "閱讀跨境網路、靜態住宅 IP 與直播運營的實用指南，內容涵蓋線路選擇、節點維護、裝置設定、支付與交付說明。文章依發佈時間更新，協助跨境電商、直播團隊與多店鋪運營者了解可用方案、常見問題與最佳實踐；如需針對特定地區或場景的建議，可透過頁面上的聯絡方式諮詢。", "Guides on cross-border networking, static residential IPs and live operations: route selection, node maintenance, device setup, payment and delivery."},
		"notice":                   {"公告", "公告", "Notices"},
		"noticeDesc":               {"查看站点公告、维护通知与服务更新，了解交付时间、线路调整、支付渠道变化和售后安排。公告按发布时间展示，涉及节点、价格或使用规则变更时以最新内容为准；建议在下单前阅读最新公告，避免维护窗口影响使用，如有疑问可通过页面联系方式咨询，我们会持续更新。", "查看站點公告、維護通知與服務更新，了解交付時間、線路調整、支付渠道變化與售後安排。公告依發佈時間展示，涉及節點、價格或使用規則變更時以最新內容為準；建議在下單前閱讀最新公告，避免維護窗口影響使用，如有疑問可透過頁面聯絡方式諮詢，我們會持續更新。", "Site notices on delivery windows, line adjustments, payment channel changes and after-sales arrangements. Read the latest notice before ordering."},
		"about":                    {"关于我们", "關於我們", "About Us"},
		"aboutDesc":                {"了解站点提供的静态住宅 IP、IEPL 专线与跨境网络方案，以及覆盖地区、适用场景、下单流程、交付方式与售后支持。页面同时说明人工交付与自动交付的区别、常见使用要求和支持渠道，帮助你在购买前确认线路、地区、设备数量与合规使用范围。如有个性化需求，可通过 Telegram 联系客服获取方案建议。", "了解站點提供的靜態住宅 IP、IEPL 專線與跨境網路方案，以及覆蓋地區、適用場景、下單流程、交付方式與售後支援。頁面同時說明人工交付與自動交付的差異、常見使用要求與支援渠道，協助你在購買前確認線路、地區、裝置數量與合規使用範圍。如有客製化需求，可透過 Telegram 聯絡客服取得方案建議。", "Static residential IPs, IEPL lines and cross-border networking: coverage, use cases, ordering, delivery and support. Confirm region, route and device limits."},
		"terms":                    {"服务条款", "服務條款", "Terms of Service"},
		"termsDesc":                {"阅读服务条款，了解服务范围、下单与支付、交付时间、退款规则、使用限制与责任说明。条款适用于本站提供的静态住宅 IP、IEPL 专线及相关网络服务；下单前请确认地区、线路、设备数量与使用场景，并遵守当地法律法规与平台规则。如对条款有疑问，建议先联系客服确认后再购买。", "閱讀服務條款，了解服務範圍、下單與支付、交付時間、退款規則、使用限制與責任說明。條款適用於本站提供的靜態住宅 IP、IEPL 專線及相關網路服務；下單前請確認地區、線路、裝置數量與使用場景，並遵守當地法律法規與平台規則。如對條款有疑問，建議先聯絡客服確認後再購買。", "Terms of service covering scope, ordering and payment, delivery windows, refunds, usage limits and liability. Confirm region, route and device count first."},
		"privacy":                  {"隐私政策", "隱私政策", "Privacy Policy"},
		"privacyDesc":              {"了解我们如何收集、使用、存储与保护你的个人信息，包括账号资料、订单信息、支付回执、必要的访问日志与设备信息。我们仅在提供服务、处理订单、保障账号与交易安全所需范围内使用这些数据，并说明 Cookie、第三方支付渠道与联系方式的处理方式；你可以通过页面上的联系方式咨询或申请相关权利。", "了解我們如何收集、使用、儲存與保護你的個人資訊，包括帳號資料、訂單資訊、支付回執、必要的存取日誌與裝置資訊。我們僅在提供服務、處理訂單、保障帳號與交易安全所需範圍內使用這些資料，並說明 Cookie、第三方支付渠道與聯絡方式的處理方式；你可以透過頁面上的聯絡方式諮詢或申請相關權利。", "How account details, orders, payment receipts, logs and device data are collected, used, stored and protected, including cookies and third-party payments."},
		"bodyRegions":              {"覆盖地区", "覆蓋地區", "Regions"},
		"bodyFeatured":             {"主推商品", "主推商品", "Featured products"},
		"bodyAllProducts":          {"全部商品", "全部商品", "All products"},
		"bodyLatestPosts":          {"最新文章", "最新文章", "Latest posts"},
		"bodyAllPosts":             {"全部文章", "全部文章", "All posts"},
		"bodyAllNotices":           {"全部公告", "全部公告", "All notices"},
		"bodyAboutServices":        {"我们的服务", "我們的服務", "Our services"},
		"bodyContact":              {"联系我们", "聯絡我們", "Contact"},
		"bodyContactTelegram":      {"通过 Telegram 联系我们", "透過 Telegram 聯絡我們", "Contact us on Telegram"},
		"bodyBlogEmpty":            {"暂无文章", "暫無文章", "No posts yet"},
		"bodyCategoryProducts":     {"分类商品", "分類商品", "Products in this category"},
		"bodyFilterNote":           {"支持按国家/地区、IP 类型与线路类型筛选。", "支援按國家/地區、IP 類型與線路類型篩選。", "Filter by region, IP type and line type."},
		"bodyProductCount":         {"共 {count} 个商品", "共 {count} 個商品", "{count} products"},
		"bodySpecPrice":            {"价格", "價格", "Price"},
		"bodySpecDelivery":         {"交付方式", "交付方式", "Delivery"},
		"bodyFulfillmentManual":    {"人工交付", "人工交付", "Manual delivery"},
		"bodyFulfillmentAuto":      {"自动交付", "自動交付", "Automatic delivery"},
		"bodyPublishedAt":          {"发布于", "發佈於", "Published"},
		"bodySpecs":                {"规格与交付", "規格與交付", "Specs & delivery"},
		"bodySpecType":             {"IP 类型", "IP 類型", "IP type"},
		"bodySpecLine":             {"线路类型", "線路類型", "Line type"},
		"bodyScenarios":            {"适用场景", "適用場景", "Use cases"},
		"bodyFaq":                  {"常见问题", "常見問題", "FAQ"},
		"bodyDetail":               {"商品详情", "商品詳情", "Details"},
		"bodyRelated":              {"相关商品", "相關商品", "Related products"},
		"bodyCoverage":             {"本页覆盖地区", "本頁覆蓋地區", "Regions on this page"},
		"bodyProcess":              {"购买与交付", "購買與交付", "Ordering & delivery"},
		"bodyStep1":                {"选择商品与规格后提交订单", "選擇商品與規格後提交訂單", "Choose a product and place the order"},
		"bodyStep2":                {"完成支付，订单进入处理", "完成付款，訂單進入處理", "Complete payment and the order is queued"},
		"bodyStep3":                {"人工交付，按订单页说明使用", "人工交付，按訂單頁說明使用", "Manual delivery with instructions on the order page"},
		"bodyStep4":                {"售后支持，按订单页联系方式处理", "售後支援，按訂單頁聯絡方式處理", "Support via the contact shown on the order page"},
		"bodySkuList":              {"可选规格", "可選規格", "Available options"},
		"bodySkuDefault":           {"默认规格", "預設規格", "Default option"},
		"lp-us-residential-ip":     {"美国静态住宅 IP", "美國靜態住宅 IP", "US Static Residential IP"},
		"lp-us-residential-ipDesc": {"美国独享静态住宅 IP，适配 TikTok 直播、跨境电商与店群运营。", "美國獨享靜態住宅 IP，適配 TikTok 直播、跨境電商與店群運營。", "Dedicated US static residential IPs for TikTok live, cross-border e-commerce and multi-store operations."},
		"lp-tiktok-live-ip":        {"TikTok 直播 IP 方案", "TikTok 直播 IP 方案", "TikTok Live IP Solutions"},
		"lp-tiktok-live-ipDesc":    {"静态住宅 IP + IEPL 中转 + 推流参数建议，降低直播掉线与卡顿。", "靜態住宅 IP + IEPL 中轉 + 推流參數建議，降低直播掉線與卡頓。", "Static residential IPs, IEPL transit and streaming guidance to reduce live drops and stutter."},
		"lp-iepl":                  {"IEPL 专线", "IEPL 專線", "IEPL Dedicated Lines"},
		"lp-ieplDesc":              {"固定路径的跨境专线，降低晚高峰延迟与丢包。", "固定路徑的跨境專線，降低晚高峰延遲與丟包。", "Fixed-path cross-border lines that reduce peak-hour latency and packet loss."},
		"lp-asia-ip":               {"亚洲住宅 IP", "亞洲住宅 IP", "Asia Residential IPs"},
		"lp-asia-ipDesc":           {"日本、台湾、菲律宾独享静态住宅 IP，按市场匹配出口。", "日本、台灣、菲律賓獨享靜態住宅 IP，按市場匹配出口。", "Dedicated Japan, Taiwan and Philippines static residential IPs matched to your market."},
		"lp-enterprise":            {"企业 IP 方案", "企業 IP 方案", "Enterprise IP Solutions"},
		"lp-enterpriseDesc":        {"多店铺、多账号统一 IP 规划、批量交付、合同发票与 SLA。", "多店鋪、多帳號統一 IP 規劃、批次交付、合約發票與 SLA。", "Unified IP planning, bulk delivery, contracts, invoices and SLA for multi-store teams."},
	}
	item, ok := texts[key]
	if !ok {
		return ""
	}
	switch locale {
	case "zh-TW":
		return item[1]
	case "en-US":
		return item[2]
	default:
		return item[0]
	}
}

// staticPageJSONLD 为关于/条款/隐私等说明页输出 WebPage + BreadcrumbList，
// 与落地页保持相同的 @graph 结构，每页只有一份 JSON-LD。
func staticPageJSONLD(base, cleanPath, title, description string, site seoSite, locale string) map[string]interface{} {
	return map[string]interface{}{
		"@context": "https://schema.org",
		"@graph": []interface{}{
			map[string]interface{}{
				"@type":       "WebPage",
				"name":        title,
				"description": description,
				"url":         base + cleanPath,
				"inLanguage":  locale,
			},
			map[string]interface{}{
				"@type": "BreadcrumbList",
				"itemListElement": []interface{}{
					map[string]interface{}{"@type": "ListItem", "position": 1, "name": site.Name, "item": base + "/"},
					map[string]interface{}{"@type": "ListItem", "position": 2, "name": title, "item": base + cleanPath},
				},
			},
		},
	}
}

// landingJSONLD 输出落地页结构化数据：WebPage + BreadcrumbList，有 FAQ 时追加 FAQPage。
// 落地页是销售页而不是商品集合页，所以不使用 CollectionPage。
func landingJSONLD(base, cleanPath string, page lpcontent.Page, site seoSite, locale string) map[string]interface{} {
	graph := []interface{}{
		map[string]interface{}{
			"@type":       "WebPage",
			"name":        page.Title,
			"description": page.Subtitle,
			"url":         base + cleanPath,
			"inLanguage":  locale,
		},
		map[string]interface{}{
			"@type": "BreadcrumbList",
			"itemListElement": []interface{}{
				map[string]interface{}{"@type": "ListItem", "position": 1, "name": site.Name, "item": base + "/"},
				map[string]interface{}{"@type": "ListItem", "position": 2, "name": page.Title, "item": base + cleanPath},
			},
		},
	}
	if len(page.Faq) > 0 {
		entities := make([]interface{}, 0, len(page.Faq))
		for _, item := range page.Faq {
			entities = append(entities, map[string]interface{}{
				"@type": "Question",
				"name":  item.Q,
				"acceptedAnswer": map[string]interface{}{
					"@type": "Answer",
					"text":  item.A,
				},
			})
		}
		graph = append(graph, map[string]interface{}{"@type": "FAQPage", "mainEntity": entities})
	}
	return map[string]interface{}{"@context": "https://schema.org", "@graph": graph}
}

// landingBodyHTML 生成不依赖 JS 的落地页正文摘要，写入 SPA 挂载点。
func landingBodyHTML(page lpcontent.Page, loc lpcontent.Locale) string {
	var b strings.Builder
	b.WriteString(`<article>`)
	fmt.Fprintf(&b, "<h1>%s</h1><p>%s</p>", html.EscapeString(page.Title), html.EscapeString(page.Subtitle))

	writeTextList(&b, loc.Sections.Pains, "ul", page.Pains)
	writeTextList(&b, loc.Sections.Solutions, "ol", page.Solutions)

	fmt.Fprintf(&b, "<h2>%s</h2><dl>", html.EscapeString(loc.Sections.Specs))
	specRows := [][2]string{
		{loc.Labels.SpecRegion, page.Specs.Region},
		{loc.Labels.SpecType, page.Specs.Type},
		{loc.Labels.SpecProtocol, page.Specs.Protocol},
		{loc.Labels.SpecDevices, page.Specs.Devices},
		{loc.Labels.SpecDelivery, page.Specs.Delivery},
		{loc.Labels.SpecBilling, page.Specs.Billing},
		{loc.Labels.SpecSupport, page.Specs.Support},
	}
	for _, row := range specRows {
		if strings.TrimSpace(row[1]) == "" {
			continue
		}
		fmt.Fprintf(&b, "<dt>%s</dt><dd>%s</dd>", html.EscapeString(row[0]), html.EscapeString(row[1]))
	}
	b.WriteString("</dl>")

	writeTextList(&b, loc.Sections.Scenarios, "ul", page.Scenarios)

	if len(page.Faq) > 0 {
		fmt.Fprintf(&b, "<h2>%s</h2>", html.EscapeString(loc.Sections.Faq))
		for _, item := range page.Faq {
			fmt.Fprintf(&b, "<h3>%s</h3><p>%s</p>", html.EscapeString(item.Q), html.EscapeString(item.A))
		}
	}

	if label := strings.TrimSpace(loc.CTA["viewProducts"]); label != "" {
		fmt.Fprintf(&b, `<p><a href="/products">%s</a></p>`, html.EscapeString(label))
	}
	b.WriteString("</article>")
	return b.String()
}

func writeTextList(b *strings.Builder, heading, tag string, items []string) {
	if len(items) == 0 {
		return
	}
	fmt.Fprintf(b, "<h2>%s</h2><%s>", html.EscapeString(heading), tag)
	for _, item := range items {
		fmt.Fprintf(b, "<li>%s</li>", html.EscapeString(item))
	}
	fmt.Fprintf(b, "</%s>", tag)
}

// ---------- 服务端正文快照（SEO） ----------
//
// SPA 首屏只有一个空的 #app，爬虫不执行 JS 时拿不到正文。这里按路由拼出
// 可索引的正文摘要写进 #app：浏览器里 Vue 挂载会整体替换，不影响交互。
// 所有来自数据库或用户的内容统一 html.EscapeString，且不输出任何 script。

const (
	homeBodyProductLimit = 3
	// homeBodyCoverageLimit 用于“覆盖地区”统计：不能只看前 3/20 个商品，
	// 否则排在后排的 US/GB 不会出现在首页正文里。
	homeBodyCoverageLimit = 200
	bodyProductLimit      = 20
	blogBodyPostLimit     = 10
	productBodyFAQLimit   = 6
	postBodyExcerptRunes  = 800
	// staticPageBodyRunes 限制后台富文本注入正文的长度，避免单页 HTML 过大。
	staticPageBodyRunes = 4000
)

// seoEnumLabels 让正文里的枚举值按语言展示；未知值原样输出，不编造翻译。
var seoEnumLabels = map[string][3]string{
	"US":          {"美国", "美國", "United States"},
	"GB":          {"英国", "英國", "United Kingdom"},
	"DE":          {"德国", "德國", "Germany"},
	"JP":          {"日本", "日本", "Japan"},
	"TW":          {"台湾", "台灣", "Taiwan"},
	"PH":          {"菲律宾", "菲律賓", "Philippines"},
	"SHARED":      {"共享节点", "共享節點", "Shared nodes"},
	"live":        {"直播", "直播", "Live streaming"},
	"ecommerce":   {"跨境电商", "跨境電商", "Cross-border e-commerce"},
	"store-group": {"店群", "店群", "Multi-store"},
	"ads":         {"广告投放", "廣告投放", "Advertising"},
	"enterprise":  {"企业方案", "企業方案", "Enterprise"},
	"test":        {"测试验证", "測試驗證", "Testing"},
	"residential": {"静态住宅", "靜態住宅", "Static residential"},
	"datacenter":  {"机房 IP", "機房 IP", "Datacenter IP"},
	"isp":         {"ISP 住宅", "ISP 住宅", "ISP"},
	"shared":      {"共享", "共享", "Shared"},
	"direct":      {"直连", "直連", "Direct"},
	"iepl":        {"IEPL 专线", "IEPL 專線", "IEPL line"},
}

func localizedTriple(item [3]string, locale string) string {
	switch locale {
	case "zh-TW":
		return item[1]
	case "en-US":
		return item[2]
	default:
		return item[0]
	}
}

func enumLabel(value, locale string) string {
	trimmed := strings.TrimSpace(value)
	item, ok := seoEnumLabels[trimmed]
	if !ok {
		return trimmed
	}
	return localizedTriple(item, locale)
}

// seoProducts 读取公开商品（仅上架）。失败时返回 nil，正文退化为不含商品的版本。
func (r *seoResolver) seoProducts(limit int, categoryID string) []productdomain.Product {
	if r == nil || r.container == nil || r.container.ProductRepo == nil || limit <= 0 {
		return nil
	}
	items, _, err := r.container.ProductRepo.List(productcontract.ListFilter{
		Page:       1,
		PageSize:   limit,
		CategoryID: categoryID,
		OnlyActive: true,
	})
	if err != nil {
		return nil
	}
	return items
}

// seoPosts 读取指定类型的已发布内容；/blog、/notice 与首页「最新文章」共用。
func (r *seoResolver) seoPosts(ctx context.Context, postType string, limit int) []contentdomain.Post {
	if r == nil || r.container == nil || r.container.ContentPostService == nil || limit <= 0 {
		return nil
	}
	posts, _, err := r.container.ContentPostService.ListPublic(ctx, contentapp.PublicPostQuery{
		Type:     postType,
		Page:     1,
		PageSize: limit,
	})
	if err != nil {
		return nil
	}
	return posts
}

func productTitle(product *productdomain.Product, locale string) string {
	title := localized(asMap(product.TitleJSON), locale)
	if title == "" {
		title = product.Slug
	}
	return title
}

func productPriceLabel(product *productdomain.Product, site seoSite) string {
	return product.PriceAmount.String() + " " + site.Currency
}

func productRegionLabel(product *productdomain.Product, locale string) string {
	return enumLabel(product.Country, locale)
}

func scenarioLabels(product *productdomain.Product, locale string) []string {
	labels := make([]string, 0, len(product.Scenario))
	for _, value := range product.Scenario {
		if label := enumLabel(value, locale); label != "" {
			labels = append(labels, label)
		}
	}
	return labels
}

func writeEscaped(b *strings.Builder, tag, value string) {
	if strings.TrimSpace(value) == "" {
		return
	}
	fmt.Fprintf(b, "<%s>%s</%s>", tag, html.EscapeString(value), tag)
}

func writeHeading(b *strings.Builder, level int, value string) {
	if strings.TrimSpace(value) == "" {
		return
	}
	fmt.Fprintf(b, "<h%d>%s</h%d>", level, html.EscapeString(value), level)
}

func writeLinkList(b *strings.Builder, heading string, items []string) {
	if len(items) == 0 {
		return
	}
	writeHeading(b, 2, heading)
	b.WriteString("<ul>")
	for _, item := range items {
		fmt.Fprintf(b, "<li>%s</li>", html.EscapeString(item))
	}
	b.WriteString("</ul>")
}

// writeProductList 输出商品条目：名称（链接）/ 价格 / 地区 / 场景。
func writeProductList(b *strings.Builder, heading, locale string, site seoSite, products []productdomain.Product) {
	if len(products) == 0 {
		return
	}
	writeHeading(b, 2, heading)
	b.WriteString("<ul>")
	for i := range products {
		product := &products[i]
		parts := []string{productPriceLabel(product, site)}
		if region := productRegionLabel(product, locale); region != "" {
			parts = append(parts, region)
		}
		if scenarios := scenarioLabels(product, locale); len(scenarios) > 0 {
			parts = append(parts, strings.Join(scenarios, " / "))
		}
		fmt.Fprintf(b, `<li><a href="/products/%s">%s</a> — %s</li>`,
			html.EscapeString(product.Slug),
			html.EscapeString(productTitle(product, locale)),
			html.EscapeString(strings.Join(parts, " · ")),
		)
	}
	b.WriteString("</ul>")
}

// writePostList 输出内容列表；basePath 为空时只输出标题（例如公告没有独立详情路由）。
func writePostList(b *strings.Builder, heading, basePath, locale, emptyText string, posts []contentdomain.Post) {
	writeHeading(b, 2, heading)
	if len(posts) == 0 {
		if strings.TrimSpace(emptyText) != "" {
			writeEscaped(b, "p", emptyText)
		}
		return
	}
	b.WriteString("<ul>")
	for i := range posts {
		post := &posts[i]
		title := localized(asMap(post.TitleJSON), locale)
		if title == "" {
			title = post.Slug
		}
		if strings.TrimSpace(basePath) == "" {
			fmt.Fprintf(b, "<li>%s", html.EscapeString(title))
		} else {
			fmt.Fprintf(b, `<li><a href="%s/%s">%s</a>`,
				html.EscapeString(strings.TrimRight(basePath, "/")),
				html.EscapeString(post.Slug),
				html.EscapeString(title),
			)
		}
		summary := truncateRunes(stripHTML(localized(asMap(post.SummaryJSON), locale)), 120)
		if summary != "" {
			fmt.Fprintf(b, " — %s", html.EscapeString(summary))
		}
		b.WriteString("</li>")
	}
	b.WriteString("</ul>")
}

func writeBreadcrumb(b *strings.Builder, siteName string, crumbs [][2]string) {
	b.WriteString(`<nav aria-label="Breadcrumb"><ol>`)
	fmt.Fprintf(b, `<li><a href="/">%s</a></li>`, html.EscapeString(siteName))
	for _, crumb := range crumbs {
		if strings.TrimSpace(crumb[1]) == "" {
			fmt.Fprintf(b, "<li>%s</li>", html.EscapeString(crumb[0]))
			continue
		}
		fmt.Fprintf(b, `<li><a href="%s">%s</a></li>`, html.EscapeString(crumb[1]), html.EscapeString(crumb[0]))
	}
	b.WriteString("</ol></nav>")
}

// ---------- 各路由正文 ----------

func (r *seoResolver) homeBodyHTML(ctx context.Context, locale, title, description string, site seoSite) string {
	b := &strings.Builder{}
	b.WriteString("<article>")
	writeHeading(b, 1, title)
	writeEscaped(b, "p", description)

	products := r.seoProducts(homeBodyCoverageLimit, "")
	featured := products
	if len(featured) > homeBodyProductLimit {
		featured = featured[:homeBodyProductLimit]
	}
	writeProductList(b, staticText("bodyFeatured", locale), locale, site, featured)
	// 覆盖地区同样来自真实在售商品，不写死国家列表。
	writeLinkList(b, staticText("bodyCoverage", locale), productCoverage(products, locale))

	// 首页没有已发布文章时整段省略，空状态只用于列表页。
	if posts := r.seoPosts(ctx, constants.PostTypeBlog, 2); len(posts) > 0 {
		writePostList(b, staticText("bodyLatestPosts", locale), "/blog", locale, "", posts)
	}

	b.WriteString(`<p><a href="/products">` + html.EscapeString(staticText("products", locale)) + `</a></p>`)
	b.WriteString("</article>")
	return b.String()
}

func (r *seoResolver) productsBodyHTML(locale, title, description string, site seoSite) string {
	b := &strings.Builder{}
	b.WriteString("<article>")
	writeHeading(b, 1, title)
	writeEscaped(b, "p", description)
	writeEscaped(b, "p", staticText("bodyFilterNote", locale))

	products := r.seoProducts(bodyProductLimit, "")
	if len(products) > 0 {
		countText := strings.ReplaceAll(staticText("bodyProductCount", locale), "{count}", strconv.Itoa(len(products)))
		writeEscaped(b, "p", countText)
	}
	writeProductList(b, staticText("bodyAllProducts", locale), locale, site, products)
	writeLinkList(b, staticText("bodyCoverage", locale), productCoverage(products, locale))
	b.WriteString("</article>")
	return b.String()
}

func (r *seoResolver) categoryBodyHTML(locale string, site seoSite, category *categorydomain.Category, description string) string {
	if category == nil {
		return ""
	}
	name := localized(asMap(category.NameJSON), locale)
	// 正文用完整分类描述（meta description 会被截断到 160 字）。
	description = firstNonEmpty(localized(asMap(category.DescriptionJSON), locale), description)
	b := &strings.Builder{}
	b.WriteString("<article>")
	writeHeading(b, 1, name)
	writeEscaped(b, "p", description)

	products := r.seoProducts(bodyProductLimit, strconv.FormatUint(uint64(category.ID), 10))
	if len(products) > 0 {
		countText := strings.ReplaceAll(staticText("bodyProductCount", locale), "{count}", strconv.Itoa(len(products)))
		writeEscaped(b, "p", countText)
	}
	writeProductList(b, staticText("bodyCategoryProducts", locale), locale, site, products)
	writeLinkList(b, staticText("bodyCoverage", locale), productCoverage(products, locale))
	b.WriteString("</article>")
	return b.String()
}

// productFAQPairs 只用商品真实字段拼 FAQ，最多 6 条；没有的字段直接跳过。
func productFAQPairs(product *productdomain.Product, locale string, site seoSite) [][2]string {
	pairs := make([][2]string, 0, productBodyFAQLimit)
	add := func(question, answer string) {
		if len(pairs) >= productBodyFAQLimit || strings.TrimSpace(answer) == "" {
			return
		}
		pairs = append(pairs, [2]string{question, answer})
	}

	add(staticText("bodySpecDelivery", locale), productDeliveryLabel(product, locale))
	add(staticText("bodySpecPrice", locale), productPriceLabel(product, site))

	if region := productRegionLabel(product, locale); region != "" {
		add(staticText("bodyRegions", locale), region)
	}
	if scenarios := scenarioLabels(product, locale); len(scenarios) > 0 {
		add(staticText("bodyScenarios", locale), strings.Join(scenarios, " / "))
	}
	if line := enumLabel(product.LineType, locale); line != "" {
		add(staticText("bodySpecLine", locale), line)
	}
	return pairs
}

func productDeliveryLabel(product *productdomain.Product, locale string) string {
	if product.FulfillmentType == constants.FulfillmentTypeAuto {
		return staticText("bodyFulfillmentAuto", locale)
	}
	return staticText("bodyFulfillmentManual", locale)
}

func (r *seoResolver) productBodyHTML(ctx context.Context, locale string, site seoSite, product *productdomain.Product) string {
	b := &strings.Builder{}
	b.WriteString("<article>")

	categorySlug := ""
	categoryName := staticText("bodyCategoryProducts", locale)
	if r != nil && r.container != nil && r.container.CategoryRepo != nil && product.CategoryID != 0 {
		if category, err := r.container.CategoryRepo.GetByID(strconv.FormatUint(uint64(product.CategoryID), 10)); err == nil && category != nil {
			categorySlug = category.Slug
			if name := localized(asMap(category.NameJSON), locale); name != "" {
				categoryName = name
			}
		}
	}
	crumbs := [][2]string{{staticText("products", locale), "/products"}}
	if categorySlug != "" {
		crumbs = append(crumbs, [2]string{categoryName, "/categories/" + categorySlug})
	}
	writeBreadcrumb(b, site.Name, crumbs)

	writeHeading(b, 1, productTitle(product, locale))
	writeEscaped(b, "p", stripHTML(localized(asMap(product.DescriptionJSON), locale)))

	writeHeading(b, 2, staticText("bodySpecs", locale))
	b.WriteString("<dl>")
	specs := [][2]string{
		{staticText("bodySpecPrice", locale), productPriceLabel(product, site)},
		{staticText("bodyRegions", locale), productRegionLabel(product, locale)},
		{staticText("bodySpecType", locale), enumLabel(product.IPType, locale)},
		{staticText("bodySpecLine", locale), enumLabel(product.LineType, locale)},
		{staticText("bodySpecDelivery", locale), productDeliveryLabel(product, locale)},
	}
	for _, row := range specs {
		if strings.TrimSpace(row[1]) == "" {
			continue
		}
		fmt.Fprintf(b, "<dt>%s</dt><dd>%s</dd>", html.EscapeString(row[0]), html.EscapeString(row[1]))
	}
	b.WriteString("</dl>")

	writeLinkList(b, staticText("bodyScenarios", locale), scenarioLabels(product, locale))

	if skus := activeSKUs(product); len(skus) > 0 {
		writeHeading(b, 2, staticText("bodySkuList", locale))
		b.WriteString("<ul>")
		for _, sku := range skus {
			label := skuLabel(sku, locale)
			fmt.Fprintf(b, `<li>%s — %s %s</li>`, html.EscapeString(label), html.EscapeString(sku.PriceAmount.String()), html.EscapeString(site.Currency))
		}
		b.WriteString("</ul>")
	}

	if pairs := productFAQPairs(product, locale, site); len(pairs) > 0 {
		writeHeading(b, 2, staticText("bodyFaq", locale))
		for _, pair := range pairs {
			writeHeading(b, 3, pair[0])
			writeEscaped(b, "p", pair[1])
		}
	}

	if content := stripHTML(localized(asMap(product.ContentJSON), locale)); content != "" {
		writeHeading(b, 2, staticText("bodyDetail", locale))
		writeEscaped(b, "p", truncateRunes(content, 1200))
	}

	// 下单到售后的真实流程，和站内结算页一致，不是编造内容。
	writeLinkList(b, staticText("bodyProcess", locale), []string{
		staticText("bodyStep1", locale),
		staticText("bodyStep2", locale),
		staticText("bodyStep3", locale),
		staticText("bodyStep4", locale),
	})

	// 同分类的其他商品：既是内部链接，也让正文覆盖更多真实商品名。
	if related := r.relatedProducts(product); len(related) > 0 {
		writeProductList(b, staticText("bodyRelated", locale), locale, site, related)
	}

	links := []string{fmt.Sprintf(`<a href="/products">%s</a>`, html.EscapeString(staticText("products", locale)))}
	if categorySlug != "" {
		links = append(links, fmt.Sprintf(`<a href="/categories/%s">%s</a>`, html.EscapeString(categorySlug), html.EscapeString(staticText("products", locale))))
	}
	if len(r.seoPosts(ctx, constants.PostTypeBlog, 2)) > 0 {
		links = append(links, fmt.Sprintf(`<a href="/blog">%s</a>`, html.EscapeString(staticText("blog", locale))))
	}
	b.WriteString("<p>" + strings.Join(links, " · ") + "</p>")
	b.WriteString("</article>")
	return b.String()
}

func (r *seoResolver) blogListBodyHTML(locale string, site seoSite, title, description string, posts []contentdomain.Post) string {
	b := &strings.Builder{}
	b.WriteString("<article>")
	// meta.Title 形如「博客 - 站点名」，正文 H1 只保留页面名。
	writeHeading(b, 1, strings.TrimSuffix(title, " - "+site.Name))
	writeEscaped(b, "p", description)
	writePostList(b, staticText("bodyAllPosts", locale), "/blog", locale, staticText("bodyBlogEmpty", locale), posts)
	b.WriteString("</article>")
	return b.String()
}

// noticeListBodyHTML 输出公告列表；公告没有独立详情路由，所以条目不外链。
func (r *seoResolver) noticeListBodyHTML(locale string, site seoSite, title, description string, posts []contentdomain.Post) string {
	b := &strings.Builder{}
	b.WriteString("<article>")
	writeHeading(b, 1, strings.TrimSuffix(title, " - "+site.Name))
	writeEscaped(b, "p", description)
	if len(posts) > 0 {
		writePostList(b, staticText("bodyAllNotices", locale), "", locale, "", posts)
	}
	b.WriteString("</article>")
	return b.String()
}

// staticPageBodyHTML 为关于/条款/隐私页生成正文。
// 优先使用后台配置的内容；未配置时用站点事实信息（真实覆盖地区、联系方式）补足，
// 不编造条款或服务承诺。
func (r *seoResolver) staticPageBodyHTML(locale, key, title, description string, site seoSite) string {
	b := &strings.Builder{}
	b.WriteString("<article>")
	if key == "about" {
		r.writeAboutBody(b, locale, title, description, site)
	} else {
		r.writeLegalBody(b, locale, key, title, description, site)
	}
	b.WriteString("</article>")
	return b.String()
}

func (r *seoResolver) writeAboutBody(b *strings.Builder, locale, title, description string, site seoSite) {
	hero := asMap(site.About["hero"])
	writeHeading(b, 1, firstNonEmpty(localized(asMap(hero["title"]), locale), title))
	writeEscaped(b, "p", firstNonEmpty(localized(asMap(hero["subtitle"]), locale), description))

	introduction := stripHTML(localized(asMap(site.About["introduction"]), locale))
	if introduction != "" {
		writeEscaped(b, "p", truncateRunes(introduction, staticPageBodyRunes))
	}

	services := asMap(site.About["services"])
	servicesTitle := localized(asMap(services["title"]), locale)
	items := localizedTextList(services["items"], locale)
	if servicesTitle != "" || len(items) > 0 {
		writeHeading(b, 2, firstNonEmpty(servicesTitle, staticText("bodyAboutServices", locale)))
		writeTextItems(b, items)
	}

	contact := asMap(site.About["contact"])
	contactTitle := localized(asMap(contact["title"]), locale)
	contactText := stripHTML(localized(asMap(contact["text"]), locale))
	if contactTitle != "" || contactText != "" {
		writeHeading(b, 2, firstNonEmpty(contactTitle, staticText("bodyContact", locale)))
		writeEscaped(b, "p", contactText)
	}

	// 后台还没填关于内容时，用真实在售地区补足正文。
	if introduction == "" && len(items) == 0 {
		writeLinkList(b, staticText("bodyCoverage", locale), productCoverage(r.seoProducts(bodyProductLimit, ""), locale))
	}
	if site.Telegram != "" {
		if contactTitle == "" && contactText == "" {
			writeHeading(b, 2, staticText("bodyContact", locale))
		}
		fmt.Fprintf(b, `<p><a href="%s" rel="noopener noreferrer">%s</a></p>`,
			html.EscapeString(site.Telegram),
			html.EscapeString(staticText("bodyContactTelegram", locale)),
		)
	}
}

func (r *seoResolver) writeLegalBody(b *strings.Builder, locale, key, title, description string, site seoSite) {
	writeHeading(b, 1, title)
	text := stripHTML(localized(asMap(site.Legal[key]), locale))
	if text != "" {
		writeTextParagraphs(b, truncateRunes(text, staticPageBodyRunes))
		return
	}
	// 后台未配置条款正文时不编造条款，只输出页面说明与联系方式。
	writeEscaped(b, "p", description)
	if site.Telegram != "" {
		fmt.Fprintf(b, `<p><a href="%s" rel="noopener noreferrer">%s</a></p>`,
			html.EscapeString(site.Telegram),
			html.EscapeString(staticText("bodyContactTelegram", locale)),
		)
	}
}

func localizedTextList(value interface{}, locale string) []string {
	items, ok := value.([]interface{})
	if !ok {
		return nil
	}
	result := make([]string, 0, len(items))
	for _, item := range items {
		text := stripHTML(localized(asMap(item), locale))
		if text != "" {
			result = append(result, truncateRunes(text, 300))
		}
	}
	return result
}

func writeTextItems(b *strings.Builder, items []string) {
	if len(items) == 0 {
		return
	}
	b.WriteString("<ul>")
	for _, item := range items {
		fmt.Fprintf(b, "<li>%s</li>", html.EscapeString(item))
	}
	b.WriteString("</ul>")
}

func writeTextParagraphs(b *strings.Builder, text string) {
	for _, line := range strings.Split(text, "\n") {
		writeEscaped(b, "p", line)
	}
}

func (r *seoResolver) postBodyHTML(locale string, site seoSite, post *contentdomain.Post) string {
	title := localized(asMap(post.TitleJSON), locale)
	if title == "" {
		title = post.Slug
	}
	b := &strings.Builder{}
	b.WriteString("<article>")
	writeBreadcrumb(b, site.Name, [][2]string{
		{staticText("blog", locale), "/blog"},
		{title, ""},
	})
	writeHeading(b, 1, title)

	published := post.CreatedAt
	if post.PublishedAt != nil {
		published = *post.PublishedAt
	}
	if !published.IsZero() {
		writeEscaped(b, "p", staticText("bodyPublishedAt", locale)+" "+published.Format("2006-01-02"))
	}

	excerpt := stripHTML(localized(asMap(post.ContentJSON), locale))
	if strings.TrimSpace(excerpt) == "" {
		excerpt = stripHTML(localized(asMap(post.SummaryJSON), locale))
	}
	writeEscaped(b, "p", truncateRunes(excerpt, postBodyExcerptRunes))
	b.WriteString("</article>")
	return b.String()
}

// relatedProducts 返回同分类的其他在售商品，最多 6 个。
func (r *seoResolver) relatedProducts(product *productdomain.Product) []productdomain.Product {
	if product == nil || product.CategoryID == 0 {
		return nil
	}
	items := r.seoProducts(bodyProductLimit, strconv.FormatUint(uint64(product.CategoryID), 10))
	related := make([]productdomain.Product, 0, 6)
	for i := range items {
		if items[i].ID == product.ID {
			continue
		}
		related = append(related, items[i])
		if len(related) == 6 {
			break
		}
	}
	return related
}

// productCoverage 返回当前页商品实际覆盖的地区（去重、保持上架顺序），不虚构未在售地区。
func productCoverage(products []productdomain.Product, locale string) []string {
	seen := make(map[string]struct{}, len(products))
	labels := make([]string, 0, len(products))
	for i := range products {
		code := strings.TrimSpace(products[i].Country)
		if code == "" {
			continue
		}
		if _, ok := seen[code]; ok {
			continue
		}
		seen[code] = struct{}{}
		labels = append(labels, enumLabel(code, locale))
	}
	return labels
}

// activeSKUs 只返回上架的 SKU；没有 SKU 的商品返回空。
func activeSKUs(product *productdomain.Product) []productdomain.ProductSKU {
	skus := make([]productdomain.ProductSKU, 0, len(product.SKUs))
	for _, sku := range product.SKUs {
		if sku.IsActive {
			skus = append(skus, sku)
		}
	}
	return skus
}

// skuLabel 用 SKU 规格值拼展示名，缺省时回落到本地化的「默认规格」。
// 键按字典序输出，保证同一 SKU 每次渲染的文案稳定。
func skuLabel(sku productdomain.ProductSKU, locale string) string {
	values := asMap(sku.SpecValuesJSON)
	if len(values) == 0 {
		return staticText("bodySkuDefault", locale)
	}
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		value := strings.TrimSpace(asString(values[key]))
		if value == "" {
			continue
		}
		parts = append(parts, value)
	}
	if len(parts) == 0 {
		return staticText("bodySkuDefault", locale)
	}
	return strings.Join(parts, " / ")
}
