package application

import (
	"context"
	"encoding/xml"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/dujiao-next/internal/modules/sitemap/contract"
	"github.com/dujiao-next/internal/modules/sitemap/domain"
)

// Service 生成 sitemap.xml / robots.txt 内容。
type Service struct {
	catalog contract.CatalogReader
	posts   contract.PublishedPostReader
	cache   contract.Cache
}

// NewService 创建 sitemap 服务。
func NewService(
	catalog contract.CatalogReader,
	posts contract.PublishedPostReader,
	cacheStore contract.Cache,
) (*Service, error) {
	if catalog == nil || posts == nil || cacheStore == nil {
		return nil, fmt.Errorf("sitemap: required dependency is nil")
	}
	return &Service{
		catalog: catalog,
		posts:   posts,
		cache:   cacheStore,
	}, nil
}

const (
	sitemapCacheTTL    = 5 * time.Minute
	sitemapCachePrefix = "sitemap:xml:"
	sitemapMaxFetch    = 50000 // 单次拉取上限，避免极端数据量打爆内存
)

// Generate 生成 sitemap.xml 内容；baseURL 必须是不带尾斜杠的站点根（如 https://example.com）
func (s *Service) Generate(ctx context.Context, baseURL string) (string, error) {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		return "", fmt.Errorf("sitemap: empty base url")
	}

	cacheKey := sitemapCachePrefix + baseURL
	if cached, err := s.cache.GetString(ctx, cacheKey); err == nil && cached != "" {
		return cached, nil
	}

	entries, err := s.collectURLs(ctx, baseURL)
	if err != nil {
		return "", err
	}

	xmlStr, err := renderSitemapXML(entries)
	if err != nil {
		return "", err
	}

	_ = s.cache.SetString(ctx, cacheKey, xmlStr, sitemapCacheTTL)
	return xmlStr, nil
}

// GenerateRobots 生成 robots.txt 内容。
// adminPath 是真实后台前缀：默认 /admin/ 始终保留，再叠加配置值，避免泄露或漏屏蔽。
func (s *Service) GenerateRobots(baseURL, adminPath string) string {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	var b strings.Builder
	b.WriteString("User-agent: *\n")
	b.WriteString("Disallow: /api/\n")
	for _, path := range robotsDisallowPaths(adminPath) {
		b.WriteString("Disallow: " + path + "\n")
	}
	b.WriteString("Disallow: /me/\n")
	b.WriteString("Disallow: /cart\n")
	b.WriteString("Disallow: /checkout\n")
	b.WriteString("Disallow: /pay\n")
	b.WriteString("Disallow: /orders/\n")
	b.WriteString("Disallow: /recharge-orders/\n")
	b.WriteString("Disallow: /guest/\n")
	b.WriteString("Disallow: /auth/\n")
	if baseURL != "" {
		b.WriteString("\n")
		b.WriteString("Sitemap: ")
		b.WriteString(baseURL)
		b.WriteString("/sitemap.xml\n")
	}
	return b.String()
}

// robotsDisallowPaths 返回需要屏蔽的后台路径：/admin/ 永远保留，
// 再叠加配置里的真实后台前缀（例如 /ythumoane/），去重且统一带尾斜杠。
func robotsDisallowPaths(adminPath string) []string {
	paths := []string{"/admin/"}
	trimmed := strings.TrimSpace(adminPath)
	if trimmed == "" {
		return paths
	}
	if !strings.HasPrefix(trimmed, "/") {
		trimmed = "/" + trimmed
	}
	normalized := strings.TrimRight(trimmed, "/") + "/"
	if normalized != "/admin/" {
		paths = append(paths, normalized)
	}
	return paths
}

// xmlEntry 是 sitemap.xml 的传输结构。
type xmlEntry struct {
	XMLName    xml.Name  `xml:"url"`
	Loc        string    `xml:"loc"`
	LastMod    string    `xml:"lastmod,omitempty"`
	ChangeFreq string    `xml:"changefreq,omitempty"`
	Priority   string    `xml:"priority,omitempty"`
	Alternates []xmlLink `xml:"xhtml:link,omitempty"`
}

// xmlLink 是 sitemap 中的 hreflang 备用链接。
type xmlLink struct {
	Rel      string `xml:"rel,attr"`
	HrefLang string `xml:"hreflang,attr"`
	Href     string `xml:"href,attr"`
}

type urlSet struct {
	XMLName    xml.Name   `xml:"urlset"`
	Xmlns      string     `xml:"xmlns,attr"`
	XmlnsXhtml string     `xml:"xmlns:xhtml,attr"`
	URLs       []xmlEntry `xml:"url"`
}

func (s *Service) collectURLs(ctx context.Context, baseURL string) ([]domain.URL, error) {
	now := time.Now().UTC().Format("2006-01-02")
	entries := make([]domain.URL, 0, 64)

	// 1. 静态页面
	// 新增 LP 必须同步这里；当前落地页来自 static lpcontent，sitemap 没有配置/数据库读取入口。
	staticPages := []struct {
		Path       string
		ChangeFreq string
		Priority   string
	}{
		{"/", "daily", "1.0"},
		{"/products", "daily", "0.9"},
		{"/blog", "weekly", "0.6"},
		{"/notice", "weekly", "0.5"},
		{"/about", "monthly", "0.3"},
		{"/terms", "yearly", "0.2"},
		{"/privacy", "yearly", "0.2"},
		{"/lp/us-residential-ip", "monthly", "0.8"},
		{"/lp/tiktok-live-ip", "monthly", "0.8"},
		{"/lp/iepl", "monthly", "0.8"},
		{"/lp/asia-ip", "monthly", "0.8"},
		{"/lp/enterprise", "monthly", "0.7"},
	}
	for _, p := range staticPages {
		entries = append(entries, domain.URL{
			Location:        baseURL + p.Path,
			LastModified:    now,
			ChangeFrequency: p.ChangeFreq,
			Priority:        p.Priority,
		})
	}

	// 2. 启用的分类
	categories, err := s.catalog.ListActiveCategories()
	if err != nil {
		return nil, fmt.Errorf("sitemap: list categories: %w", err)
	}
	for _, cat := range categories {
		entries = append(entries, domain.URL{
			Location:        baseURL + "/categories/" + url.PathEscape(cat.Slug),
			LastModified:    cat.CreatedAt.UTC().Format("2006-01-02"),
			ChangeFrequency: "weekly",
			Priority:        "0.7",
		})
	}

	// 3. 上架的商品（OnlyActive 已含分类启用过滤）
	products, err := s.catalog.ListActiveProducts(sitemapMaxFetch)
	if err != nil {
		return nil, fmt.Errorf("sitemap: list products: %w", err)
	}
	for _, p := range products {
		entries = append(entries, domain.URL{
			Location:        baseURL + "/products/" + url.PathEscape(p.Slug),
			LastModified:    p.UpdatedAt.UTC().Format("2006-01-02"),
			ChangeFrequency: "daily",
			Priority:        "0.8",
		})
	}

	// 4. 已发布的博客 / 公告
	posts, err := s.posts.ListPublishedPosts(ctx, sitemapMaxFetch)
	if err != nil {
		return nil, fmt.Errorf("sitemap: list posts: %w", err)
	}
	for _, post := range posts {
		lastmod := post.CreatedAt
		if post.PublishedAt != nil {
			lastmod = *post.PublishedAt
		}
		// blog 与 notice 共用 /blog/:slug 详情页（user 前台 Notice.vue 跳转到 /blog/{slug}）
		entries = append(entries, domain.URL{
			Location:        baseURL + "/blog/" + url.PathEscape(post.Slug),
			LastModified:    lastmod.UTC().Format("2006-01-02"),
			ChangeFrequency: "monthly",
			Priority:        "0.5",
		})
	}

	// 为每个可索引 URL 生成三语 hreflang；前台 detectLocale 会读取 ?lang=。
	for i := range entries {
		path := strings.TrimPrefix(entries[i].Location, baseURL)
		if path == "" {
			path = "/"
		}
		entries[i].Alternates = map[string]string{
			"zh-CN":     baseURL + path + "?lang=zh-CN",
			"zh-TW":     baseURL + path + "?lang=zh-TW",
			"en-US":     baseURL + path + "?lang=en-US",
			"x-default": baseURL + path,
		}
	}

	return entries, nil
}

func renderSitemapXML(entries []domain.URL) (string, error) {
	xmlEntries := make([]xmlEntry, 0, len(entries))
	for _, entry := range entries {
		links := make([]xmlLink, 0, len(entry.Alternates))
		for lang, href := range entry.Alternates {
			if strings.TrimSpace(lang) == "" || strings.TrimSpace(href) == "" {
				continue
			}
			links = append(links, xmlLink{Rel: "alternate", HrefLang: lang, Href: href})
		}
		xmlEntries = append(xmlEntries, xmlEntry{
			Loc:        entry.Location,
			LastMod:    entry.LastModified,
			ChangeFreq: entry.ChangeFrequency,
			Priority:   entry.Priority,
			Alternates: links,
		})
	}
	set := urlSet{
		Xmlns:      "http://www.sitemaps.org/schemas/sitemap/0.9",
		XmlnsXhtml: "http://www.w3.org/1999/xhtml",
		URLs:       xmlEntries,
	}
	body, err := xml.MarshalIndent(set, "", "  ")
	if err != nil {
		return "", err
	}
	return xml.Header + string(body) + "\n", nil
}
