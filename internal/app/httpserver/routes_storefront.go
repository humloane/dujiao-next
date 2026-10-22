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
	Currency    string
	Telegram    string
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
		site.Image = image
	}
	if currency := asString(raw[constants.SettingFieldSiteCurrency]); currency != "" {
		site.Currency = currency
	}
	site.Telegram = asString(asMap(raw["contact"])["telegram"])
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
		Image:       absoluteAssetURL(base, site.Image),
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
		meta.BodyHTML = simpleBodyHTML(site.Name, meta.Description)
	case cleanPath == "/products":
		meta.Title = joinTitle(staticText("products", locale), site.Name)
		meta.Description = firstNonEmpty(localized(site.SeoDesc, locale), staticText("productsDesc", locale), meta.Description)
		meta.JSONLD = marshalJSONLD(collectionJSONLD(base, cleanPath, meta.Title, meta.Description, site, locale))
		meta.BodyHTML = simpleBodyHTML(staticText("products", locale), meta.Description)
	case strings.HasPrefix(cleanPath, "/categories/"):
		slug := strings.TrimPrefix(cleanPath, "/categories/")
		if category := r.findCategory(slug); category != nil {
			name := localized(asMap(category.NameJSON), locale)
			seo := asMap(category.SeoMetaJSON)
			meta.Title = joinTitle(firstNonEmpty(asStringFromMap(seo["title"], locale), name), site.Name)
			meta.Description = truncateRunes(firstNonEmpty(
				asStringFromMap(seo["description"], locale),
				localized(asMap(category.DescriptionJSON), locale),
				meta.Description,
				staticText("productsDesc", locale),
			), seoDescriptionMaxRunes)
			meta.Keywords = asStringFromMap(seo["keywords"], locale)
			meta.JSONLD = marshalJSONLD(collectionJSONLD(base, cleanPath, meta.Title, meta.Description, site, locale))
			meta.BodyHTML = simpleBodyHTML(name, meta.Description)
		}
	case strings.HasPrefix(cleanPath, "/products/"):
		slug := strings.TrimPrefix(cleanPath, "/products/")
		if product := r.findProduct(slug); product != nil {
			r.applyProductMeta(meta, base, cleanPath, locale, site, product)
		}
	case strings.HasPrefix(cleanPath, "/blog/"):
		slug := strings.TrimPrefix(cleanPath, "/blog/")
		if post := r.findPost(ctx, slug); post != nil {
			r.applyPostMeta(meta, base, cleanPath, locale, site, post)
		}
	case cleanPath == "/blog" || cleanPath == "/notice":
		key := strings.TrimPrefix(cleanPath, "/")
		meta.Title = joinTitle(staticText(key, locale), site.Name)
		meta.Description = firstNonEmpty(staticText(key+"Desc", locale), meta.Description)
		var titles []string
		if cleanPath == "/blog" {
			titles = r.listPostTitles(ctx, locale, 12)
		}
		// 没有已发布文章时不要输出空的 CollectionPage。
		if cleanPath != "/blog" || len(titles) > 0 {
			meta.JSONLD = marshalJSONLD(collectionJSONLD(base, cleanPath, meta.Title, meta.Description, site, locale))
		}
		meta.BodyHTML = listBodyHTML(meta.Title, meta.Description, titles)
	case cleanPath == "/about" || cleanPath == "/terms" || cleanPath == "/privacy":
		key := strings.TrimPrefix(cleanPath, "/")
		meta.Title = joinTitle(staticText(key, locale), site.Name)
		meta.Description = firstNonEmpty(staticText(key+"Desc", locale), meta.Description)
		meta.BodyHTML = simpleBodyHTML(staticText(key, locale), meta.Description)
	case strings.HasPrefix(cleanPath, "/lp/"):
		slug := strings.TrimPrefix(cleanPath, "/lp/")
		page, loc, ok := lpcontent.Lookup(locale, slug)
		if !ok {
			meta.Robots = "noindex, nofollow"
			break
		}
		// 落地页是独立销售页：用 WebPage 而不是 CollectionPage，
		// 关键词也来自落地页自己的文案，不复用站点 keywords。
		meta.Title = joinTitle(page.Title, site.Name)
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

func (r *seoResolver) applyProductMeta(meta *web.PageMeta, base, cleanPath, locale string, site seoSite, product *productdomain.Product) {
	seo := asMap(product.SeoMetaJSON)
	title := localized(asMap(product.TitleJSON), locale)
	if title == "" {
		title = product.Slug
	}
	meta.Title = joinTitle(firstNonEmpty(asStringFromMap(seo["title"], locale), title), site.Name)
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
	jsonLD := map[string]interface{}{
		"@context":    "https://schema.org",
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
		jsonLD["image"] = images
	}
	meta.JSONLD = marshalJSONLD(jsonLD)
	meta.BodyHTML = simpleBodyHTML(title, meta.Description)
}

func (r *seoResolver) applyPostMeta(meta *web.PageMeta, base, cleanPath, locale string, site seoSite, post *contentdomain.Post) {
	title := localized(asMap(post.TitleJSON), locale)
	if title == "" {
		title = post.Slug
	}
	meta.Title = joinTitle(title, site.Name)
	meta.Description = truncateRunes(firstNonEmpty(
		stripHTML(localized(asMap(post.SummaryJSON), locale)),
		stripHTML(localized(asMap(post.ContentJSON), locale)),
		localized(site.SeoDesc, locale),
	), seoDescriptionMaxRunes)
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
	meta.BodyHTML = simpleBodyHTML(title, meta.Description)
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
	if site == "" || page == site || strings.Contains(page, site) {
		return page
	}
	return page + " - " + site
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
		"products":                 {"产品中心", "產品中心", "Products"},
		"productsDesc":             {"浏览全部静态住宅 IP、IEPL 专线与跨境网络方案。", "瀏覽全部靜態住宅 IP、IEPL 專線與跨境網路方案。", "Browse static residential IPs, IEPL lines and cross-border networking plans."},
		"blog":                     {"博客", "部落格", "Blog"},
		"blogDesc":                 {"跨境网络、静态住宅 IP 与直播运营的实用指南。", "跨境網路、靜態住宅 IP 與直播運營的實用指南。", "Practical guides on cross-border networking, static residential IPs and live operations."},
		"notice":                   {"公告", "公告", "Notices"},
		"noticeDesc":               {"站点公告与服务更新。", "站點公告與服務更新。", "Site announcements and service updates."},
		"about":                    {"关于我们", "關於我們", "About Us"},
		"aboutDesc":                {"了解我们的团队、服务与交付方式。", "了解我們的團隊、服務與交付方式。", "Learn about our team, services and delivery."},
		"terms":                    {"服务条款", "服務條款", "Terms of Service"},
		"termsDesc":                {"服务范围、交付、退款与使用规则。", "服務範圍、交付、退款與使用規則。", "Service scope, delivery, refunds and usage rules."},
		"privacy":                  {"隐私政策", "隱私政策", "Privacy Policy"},
		"privacyDesc":              {"我们如何收集、使用与保护你的个人信息。", "我們如何收集、使用與保護你的個人資訊。", "How we collect, use and protect your personal information."},
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

// listPostTitles 返回最近的已发布文章标题，用于 /blog 的正文摘要与非空判断。
func (r *seoResolver) listPostTitles(ctx context.Context, locale string, limit int) []string {
	if r == nil || r.container == nil || r.container.ContentPostService == nil {
		return nil
	}
	posts, _, err := r.container.ContentPostService.ListPublic(ctx, contentapp.PublicPostQuery{Page: 1, PageSize: limit})
	if err != nil {
		return nil
	}
	titles := make([]string, 0, len(posts))
	for _, post := range posts {
		title := localized(asMap(post.TitleJSON), locale)
		if strings.TrimSpace(title) == "" {
			title = post.Slug
		}
		titles = append(titles, title)
	}
	return titles
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

// simpleBodyHTML 是详情页/静态页的最小正文摘要：标题 + 描述。
func simpleBodyHTML(title, description string) string {
	title = strings.TrimSpace(title)
	description = strings.TrimSpace(description)
	if title == "" && description == "" {
		return ""
	}
	var b strings.Builder
	b.WriteString("<article>")
	if title != "" {
		fmt.Fprintf(&b, "<h1>%s</h1>", html.EscapeString(title))
	}
	if description != "" {
		fmt.Fprintf(&b, "<p>%s</p>", html.EscapeString(description))
	}
	b.WriteString("</article>")
	return b.String()
}

// listBodyHTML 用于列表页：标题 + 描述 + 若干条目链接文案。
func listBodyHTML(title, description string, items []string) string {
	var b strings.Builder
	b.WriteString("<article>")
	if title != "" {
		fmt.Fprintf(&b, "<h1>%s</h1>", html.EscapeString(title))
	}
	if description != "" {
		fmt.Fprintf(&b, "<p>%s</p>", html.EscapeString(description))
	}
	if len(items) > 0 {
		b.WriteString("<ul>")
		for _, item := range items {
			fmt.Fprintf(&b, "<li>%s</li>", html.EscapeString(item))
		}
		b.WriteString("</ul>")
	}
	b.WriteString("</article>")
	return b.String()
}
