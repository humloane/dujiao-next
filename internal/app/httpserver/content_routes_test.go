package httpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"github.com/dujiao-next/internal/app/httpserver/middleware"
	"github.com/dujiao-next/internal/authz"
	"github.com/dujiao-next/internal/constants"
	productdomain "github.com/dujiao-next/internal/modules/catalog/product/domain"
	contentdomain "github.com/dujiao-next/internal/modules/content/domain"
	contenttransport "github.com/dujiao-next/internal/modules/content/transport/http"
	admincontract "github.com/dujiao-next/internal/modules/identity/admin/contract"
	admindomain "github.com/dujiao-next/internal/modules/identity/admin/domain"
	adminstore "github.com/dujiao-next/internal/modules/identity/admin/infrastructure/gormstore"
	adminauthapp "github.com/dujiao-next/internal/modules/identity/adminauth/application"
	"github.com/dujiao-next/internal/modules/identity/jwttoken"
	"github.com/dujiao-next/internal/platform/http/response"
	"github.com/dujiao-next/internal/shared/jsonmap"
	"github.com/dujiao-next/internal/shared/money"
	"github.com/dujiao-next/internal/web"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
)

const contentAdminJWTSecret = "content-admin-route-contract-secret"

var contentAdminRouteContract = []adminRoute{
	{method: http.MethodGet, object: "/admin/posts"},
	{method: http.MethodPost, object: "/admin/posts"},
	{method: http.MethodPut, object: "/admin/posts/:id"},
	{method: http.MethodDelete, object: "/admin/posts/:id"},
	{method: http.MethodGet, object: "/admin/posts/:id/products"},
	{method: http.MethodGet, object: "/admin/post-categories"},
	{method: http.MethodPost, object: "/admin/post-categories"},
	{method: http.MethodPut, object: "/admin/post-categories/:id"},
	{method: http.MethodDelete, object: "/admin/post-categories/:id"},
	{method: http.MethodPatch, object: "/admin/post-categories/:id/status"},
	{method: http.MethodGet, object: "/admin/banners"},
	{method: http.MethodGet, object: "/admin/banners/:id"},
	{method: http.MethodPost, object: "/admin/banners"},
	{method: http.MethodPut, object: "/admin/banners/:id"},
	{method: http.MethodDelete, object: "/admin/banners/:id"},
	{method: http.MethodGet, object: "/admin/media"},
	{method: http.MethodPost, object: "/admin/media/batch-delete"},
	{method: http.MethodPut, object: "/admin/media/:id"},
	{method: http.MethodDelete, object: "/admin/media/:id"},
}

var contentPublicRouteContract = []adminRoute{
	{method: http.MethodGet, object: "/api/v1/public/posts"},
	{method: http.MethodGet, object: "/api/v1/public/posts/:slug"},
	{method: http.MethodGet, object: "/api/v1/public/banners"},
	{method: http.MethodGet, object: "/api/v1/public/post-categories"},
}

func TestContentRouteContractMatchesRouterSource(t *testing.T) {
	adminRoutes, err := extractAdminRoutesFromSource()
	if err != nil {
		t.Fatalf("extract admin routes: %v", err)
	}
	actualAdmin := make([]adminRoute, 0, len(contentAdminRouteContract))
	for _, route := range adminRoutes {
		if isContentAdminRoute(route.object) {
			actualAdmin = append(actualAdmin, route)
		}
	}
	assertRouteSetsEqual(t, contentAdminRouteContract, actualAdmin)

	publicRoutes, err := extractPublicContentRoutesFromSource()
	if err != nil {
		t.Fatalf("extract public content routes: %v", err)
	}
	assertRouteSetsEqual(t, contentPublicRouteContract, publicRoutes)
}

func TestContentRoutesStayInExpectedRouterTrustGroups(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve content route test filename")
	}
	routerDirectory := filepath.Dir(thisFile)
	registrations := []struct {
		file      string
		statement string
	}{
		{file: "routes_storefront.go", statement: "contenttransport.RegisterPublicRoutes(public, publicContentHandler)"},
		{file: "routes_admin.go", statement: "contenttransport.RegisterAdminRoutes(authorized, adminContentHandler)"},
	}

	for _, registration := range registrations {
		raw, err := os.ReadFile(filepath.Join(routerDirectory, registration.file))
		if err != nil {
			t.Fatalf("read production route file %s: %v", registration.file, err)
		}
		source := string(raw)
		if !strings.Contains(source, registration.statement) {
			t.Errorf("%s must contain %q; changing the route group can bypass the intended authentication or public boundary", registration.file, registration.statement)
		}
	}
}

func TestAdminContentRoutesRequireAuthenticationAndOperationsPermission(t *testing.T) {
	adminRepo, authzService, noPermissionToken, operationsToken := setupContentRouteAccessTest(t)

	tests := []struct {
		name       string
		token      string
		wantStatus int
	}{
		{name: "anonymous", wantStatus: response.CodeUnauthorized},
		{name: "invalid token", token: "not-a-jwt", wantStatus: response.CodeUnauthorized},
		{name: "no permission", token: noPermissionToken, wantStatus: response.CodeForbidden},
		{name: "operations", token: operationsToken, wantStatus: response.CodeOK},
	}

	for _, route := range contentAdminRouteContract {
		route := route
		t.Run(route.method+" "+route.object, func(t *testing.T) {
			for _, test := range tests {
				t.Run(test.name, func(t *testing.T) {
					router := newContentRouteAccessRouter(adminRepo, authzService)
					path := strings.ReplaceAll("/api/v1"+route.object, ":id", "42")
					request := httptest.NewRequest(route.method, path, nil)
					if test.token != "" {
						request.Header.Set("Authorization", "Bearer "+test.token)
					}
					recorder := httptest.NewRecorder()
					router.ServeHTTP(recorder, request)

					var got struct {
						StatusCode int `json:"status_code"`
					}
					if err := json.Unmarshal(recorder.Body.Bytes(), &got); err != nil {
						t.Fatalf("decode response: %v body=%s", err, recorder.Body.String())
					}
					if got.StatusCode != test.wantStatus {
						t.Fatalf("business status want %d got %d body=%s", test.wantStatus, got.StatusCode, recorder.Body.String())
					}
				})
			}
		})
	}
}

func extractPublicContentRoutesFromSource() ([]adminRoute, error) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		return nil, fmt.Errorf("resolve content route test filename")
	}
	routerSource := filepath.Join(filepath.Dir(thisFile), "..", "..", "modules", "content", "transport", "http", "routes.go")
	raw, err := os.ReadFile(routerSource)
	if err != nil {
		return nil, err
	}

	re := regexp.MustCompile(`public\.(GET|POST|PUT|PATCH|DELETE)\("([^"]+)"`)
	matches := re.FindAllStringSubmatch(string(raw), -1)
	routes := make([]adminRoute, 0, len(contentPublicRouteContract))
	for _, match := range matches {
		path := "/api/v1/public" + match[2]
		if isContentPublicRoute(path) {
			routes = append(routes, adminRoute{method: match[1], object: path})
		}
	}
	return routes, nil
}

func isContentAdminRoute(object string) bool {
	return object == "/admin/posts" || strings.HasPrefix(object, "/admin/posts/") ||
		object == "/admin/post-categories" || strings.HasPrefix(object, "/admin/post-categories/") ||
		object == "/admin/banners" || strings.HasPrefix(object, "/admin/banners/") ||
		object == "/admin/media" || strings.HasPrefix(object, "/admin/media/")
}

func isContentPublicRoute(path string) bool {
	return path == "/api/v1/public/posts" || path == "/api/v1/public/posts/:slug" ||
		path == "/api/v1/public/banners" || path == "/api/v1/public/post-categories"
}

func assertRouteSetsEqual(t *testing.T, want, got []adminRoute) {
	t.Helper()
	normalize := func(routes []adminRoute) []string {
		result := make([]string, 0, len(routes))
		for _, route := range routes {
			result = append(result, route.method+" "+route.object)
		}
		sort.Strings(result)
		return result
	}
	wantRoutes := normalize(want)
	gotRoutes := normalize(got)
	if strings.Join(wantRoutes, "\n") != strings.Join(gotRoutes, "\n") {
		t.Fatalf("content route contract mismatch\nwant:\n%s\ngot:\n%s", strings.Join(wantRoutes, "\n"), strings.Join(gotRoutes, "\n"))
	}
}

func setupContentRouteAccessTest(t *testing.T) (admincontract.Store, *authz.Service, string, string) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	dsn := fmt.Sprintf("file:content_route_access_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&admindomain.Admin{}); err != nil {
		t.Fatalf("auto migrate admins: %v", err)
	}

	adminRepo := adminstore.New(db)
	noPermission := &admindomain.Admin{Username: "content-none", PasswordHash: "hash"}
	operations := &admindomain.Admin{Username: "content-operations", PasswordHash: "hash"}
	if err := adminRepo.Create(noPermission); err != nil {
		t.Fatalf("create no-permission admin: %v", err)
	}
	if err := adminRepo.Create(operations); err != nil {
		t.Fatalf("create operations admin: %v", err)
	}

	authzService, err := authz.NewService(db)
	if err != nil {
		t.Fatalf("create authz service: %v", err)
	}
	if err := authzService.BootstrapBuiltinRoles(); err != nil {
		t.Fatalf("bootstrap builtin roles: %v", err)
	}
	if err := authzService.SetAdminRoles(operations.ID, []string{"operations"}); err != nil {
		t.Fatalf("assign operations role: %v", err)
	}

	return adminRepo, authzService,
		signContentAdminToken(t, noPermission),
		signContentAdminToken(t, operations)
}

func signContentAdminToken(t *testing.T, admin *admindomain.Admin) string {
	t.Helper()
	now := time.Now()
	claims := adminauthapp.JWTClaims{
		AdminID:      admin.ID,
		Username:     admin.Username,
		TokenVersion: admin.TokenVersion,
		Typ:          jwttoken.TypeAccess,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now.Add(-time.Second)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(contentAdminJWTSecret))
	if err != nil {
		t.Fatalf("sign admin token: %v", err)
	}
	return signed
}

func newContentRouteAccessRouter(adminRepo admincontract.Store, authzService *authz.Service) *gin.Engine {
	router := gin.New()
	admin := router.Group("/api/v1/admin")
	admin.Use(middleware.JWTAuthMiddleware(contentAdminJWTSecret, adminRepo), middleware.AdminRBACMiddleware(authzService))
	admin.Use(func(c *gin.Context) {
		response.Success(c, nil)
		c.Abort()
	})
	contenttransport.RegisterAdminRoutes(admin, contenttransport.NewAdminHandler(nil, nil, nil, nil))
	return router
}
func countH1(body string) int {
	return strings.Count(body, "<h1>")
}

// 详情页正文只输出真实可推导的字段：库存来自 SKU/卡密汇总，列表查询拿不到，
// 因此不能拿 product.ManualStockTotal 冒充库存，更不能绕过库存展示模式。
func TestProductBodyHTMLOmitsStockAndEscapesContent(t *testing.T) {
	product := &productdomain.Product{
		Slug:             "test-product",
		TitleJSON:        jsonmap.JSON{"zh-CN": `测试 <script>alert(1)</script>`},
		DescriptionJSON:  jsonmap.JSON{"zh-CN": "描述 <b>加粗</b>"},
		PriceAmount:      money.FromDecimal(decimal.NewFromInt(10)),
		Country:          "US",
		IPType:           productdomain.IPTypeResidential,
		LineType:         productdomain.LineTypeIEPL,
		FulfillmentType:  constants.FulfillmentTypeManual,
		ManualStockTotal: 7,
	}

	body := (&seoResolver{}).productBodyHTML(context.Background(), "zh-CN", seoSite{Name: "站点", Currency: "CNY"}, product)

	if got := countH1(body); got != 1 {
		t.Fatalf("expected exactly one h1, got %d: %s", got, body)
	}
	if strings.Contains(body, "<script") {
		t.Fatalf("body must not contain raw script tags: %s", body)
	}
	if !strings.Contains(body, "&lt;script&gt;") {
		t.Fatalf("title should be html-escaped: %s", body)
	}
	if strings.Contains(body, "库存") || !strings.Contains(body, "交付方式") {
		t.Fatalf("specs must keep delivery info without stock numbers: %s", body)
	}
}

// /notice 与 /blog 共用标题文案，但不能复用博客列表正文，否则会输出
// 「全部文章 / 暂无文章」这种与页面无关的内容。
func TestNoticeBodyDoesNotRenderBlogList(t *testing.T) {
	meta := (&seoResolver{}).buildMeta(web.MetaRequest{Path: "/notice"}, "zh-CN", seoSite{
		Name:     "站点",
		URL:      "https://example.com",
		Currency: "CNY",
	})

	if !strings.Contains(meta.BodyHTML, "<h1>公告</h1>") {
		t.Fatalf("notice body should have its own h1: %s", meta.BodyHTML)
	}
	if strings.Contains(meta.BodyHTML, "全部文章") || strings.Contains(meta.BodyHTML, "暂无文章") {
		t.Fatalf("notice body must not reuse the blog list: %s", meta.BodyHTML)
	}
}

// 首页没有文章时整段省略，不输出空状态。
func TestHomeBodyHTMLEmitsNoEmptyPostSection(t *testing.T) {
	body := (&seoResolver{}).homeBodyHTML(context.Background(), "zh-CN", "站点", "一句话介绍", seoSite{
		Name:     "站点",
		Currency: "CNY",
	})

	if got := countH1(body); got != 1 {
		t.Fatalf("expected exactly one h1, got %d: %s", got, body)
	}
	if strings.Contains(body, "最新文章") || strings.Contains(body, "暂无文章") {
		t.Fatalf("home body should omit the post section when there are no posts: %s", body)
	}
}

// 列表页标题不能用「主推商品」，这个说法只适用于首页精选。
func TestProductsBodyHTMLUsesListHeading(t *testing.T) {
	body := (&seoResolver{}).productsBodyHTML("zh-CN", "产品中心", "描述", seoSite{Name: "站点", Currency: "CNY"})

	if got := countH1(body); got != 1 {
		t.Fatalf("expected exactly one h1, got %d: %s", got, body)
	}
	if strings.Contains(body, "主推商品") {
		t.Fatalf("list page must not label products as featured: %s", body)
	}
}

// 关于/条款/隐私页需要服务端 WebPage + BreadcrumbList，且每页只有一份。
func TestStaticPagesEmitWebPageJSONLD(t *testing.T) {
	for _, path := range []string{"/about", "/terms", "/privacy"} {
		meta := (&seoResolver{}).buildMeta(web.MetaRequest{Path: path}, "zh-CN", seoSite{
			Name:     "站点",
			URL:      "https://example.com",
			Currency: "CNY",
		})
		if meta.JSONLD == "" {
			t.Fatalf("%s: expected JSON-LD", path)
		}
		if !strings.Contains(meta.JSONLD, `"WebPage"`) || !strings.Contains(meta.JSONLD, `"BreadcrumbList"`) {
			t.Fatalf("%s: unexpected JSON-LD: %s", path, meta.JSONLD)
		}
		if strings.Count(meta.JSONLD, `"@context"`) != 1 {
			t.Fatalf("%s: expected a single @context, got %s", path, meta.JSONLD)
		}
	}
}

// 静态页正文优先使用后台配置内容（关于/条款/隐私）。
func TestStaticPageBodyUsesConfiguredContent(t *testing.T) {
	site := seoSite{
		Name:     "站点",
		URL:      "https://example.com",
		Currency: "CNY",
		Telegram: "https://t.me/example",
		About: map[string]interface{}{
			"hero": map[string]interface{}{
				"title":    map[string]interface{}{"zh-CN": "关于我们"},
				"subtitle": map[string]interface{}{"zh-CN": "副标题"},
			},
			"introduction": map[string]interface{}{"zh-CN": "<p>我们提供真实住宅 IP。</p>"},
			"services": map[string]interface{}{
				"title": map[string]interface{}{"zh-CN": "服务"},
				"items": []interface{}{map[string]interface{}{"zh-CN": "静态住宅 IP"}},
			},
			"contact": map[string]interface{}{
				"title": map[string]interface{}{"zh-CN": "联系"},
				"text":  map[string]interface{}{"zh-CN": "Telegram 客服"},
			},
		},
		Legal: map[string]interface{}{
			"terms": map[string]interface{}{"zh-CN": "条款第一段\n条款第二段"},
		},
	}

	about := (&seoResolver{}).buildMeta(web.MetaRequest{Path: "/about"}, "zh-CN", site)
	if got := countH1(about.BodyHTML); got != 1 {
		t.Fatalf("about: expected one h1, got %d: %s", got, about.BodyHTML)
	}
	for _, want := range []string{"副标题", "我们提供真实住宅 IP。", "静态住宅 IP", "Telegram 客服"} {
		if !strings.Contains(about.BodyHTML, want) {
			t.Fatalf("about body missing %q: %s", want, about.BodyHTML)
		}
	}

	terms := (&seoResolver{}).buildMeta(web.MetaRequest{Path: "/terms"}, "zh-CN", site)
	if got := countH1(terms.BodyHTML); got != 1 {
		t.Fatalf("terms: expected one h1, got %d: %s", got, terms.BodyHTML)
	}
	for _, want := range []string{"条款第一段", "条款第二段"} {
		if !strings.Contains(terms.BodyHTML, want) {
			t.Fatalf("terms body missing %q: %s", want, terms.BodyHTML)
		}
	}
}

// 公告列表不能链到 /blog，也不复用博客标题。
func TestNoticeListBodyDoesNotLinkToBlog(t *testing.T) {
	posts := []contentdomain.Post{
		{Slug: "notice-1", TitleJSON: jsonmap.JSON{"zh-CN": "公告一"}},
		{Slug: "notice-2", TitleJSON: jsonmap.JSON{"zh-CN": "公告二"}},
	}
	body := (&seoResolver{}).noticeListBodyHTML("zh-CN", seoSite{Name: "站点"}, "公告 - 站点", "站点公告", posts)

	if !strings.Contains(body, "<h1>公告</h1>") || !strings.Contains(body, "全部公告") {
		t.Fatalf("unexpected notice body: %s", body)
	}
	if strings.Contains(body, "/blog/") {
		t.Fatalf("notice list must not link to blog: %s", body)
	}
	if !strings.Contains(body, "公告一") || !strings.Contains(body, "公告二") {
		t.Fatalf("notice titles missing: %s", body)
	}
}
