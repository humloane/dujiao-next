package web

import (
	"bytes"
	"context"
	"html"
	"net/http"
	"strings"
	"sync"
)

// PageMeta 是服务端注入到 SPA 入口 HTML 的 SEO 元数据。
//
// 本项目采用单二进制 embed，构建期拿不到线上数据库内容，因此不做静态
// 预渲染，而是在返回 index.html 时按路径实时注入 title/description/OG/
// hreflang/JSON-LD，并把当前页面可公开的正文摘要写进 #app（BodyHTML）。
// 搜索引擎与社交爬虫无需执行 JS 即可拿到真实元数据与正文关键词；
// 浏览器里 Vue 挂载会替换 #app 内容，正常用户看到的仍是同一个 SPA。
type PageMeta struct {
	Title       string
	Description string
	Keywords    string
	Canonical   string
	Image       string
	Type        string
	Locale      string
	Robots      string
	JSONLD      string
	Alternates  map[string]string
	// Path 是规范化后的站点路径（如 /lp/iepl），用于标记服务端 JSON-LD，
	// 前端路由切换时据此判断是否需要自己再插入一份。
	Path string
	// BodyHTML 是不依赖 JS 的正文摘要，写入 <div id="app"> 内。
	BodyHTML string
}

// MetaRequest 是解析页面元数据所需的请求上下文。
type MetaRequest struct {
	Context        context.Context
	Path           string
	RawQuery       string
	Host           string
	Scheme         string
	AcceptLanguage string
}

// MetaProvider 根据请求返回页面元数据；返回 nil 表示不注入。
type MetaProvider func(MetaRequest) *PageMeta

var (
	metaMu       sync.RWMutex
	metaProvider MetaProvider
)

// SetMetaProvider 注册全局元数据解析器。应用启动时调用一次；测试可传 nil 复位。
func SetMetaProvider(provider MetaProvider) {
	metaMu.Lock()
	defer metaMu.Unlock()
	metaProvider = provider
}

func getMetaProvider() MetaProvider {
	metaMu.RLock()
	defer metaMu.RUnlock()
	return metaProvider
}

// injectMetaHTML 把 PageMeta 注入 SPA 入口 HTML。
//
// 只做字符串级替换，不引入 HTML 解析器：入口模板由本仓库控制，结构稳定；
// 所有来自数据库的值统一经过 html.EscapeString，JSON-LD 额外转义 "</"，
// 防止内容里的 </script> 提前闭合脚本标签。
func injectMetaHTML(index []byte, meta *PageMeta) []byte {
	if meta == nil || strings.TrimSpace(meta.Title) == "" {
		return index
	}

	locale := strings.TrimSpace(meta.Locale)
	if locale == "" {
		locale = "zh-CN"
	}
	result := bytes.Replace(index, []byte(`<html lang="en">`), []byte(`<html lang="`+html.EscapeString(locale)+`">`), 1)

	titleTag := []byte("<title></title>")
	newTitle := []byte("<title>" + html.EscapeString(strings.TrimSpace(meta.Title)) + "</title>")
	if bytes.Contains(result, titleTag) {
		result = bytes.Replace(result, titleTag, newTitle, 1)
	} else {
		result = bytes.Replace(result, []byte("</head>"), append(newTitle, []byte("</head>")...), 1)
	}

	var head strings.Builder
	writeMetaTag := func(name, content string) {
		if strings.TrimSpace(content) == "" {
			return
		}
		head.WriteString(`<meta name="` + html.EscapeString(name) + `" content="` + html.EscapeString(content) + `">`)
	}
	writeProperty := func(property, content string) {
		if strings.TrimSpace(content) == "" {
			return
		}
		head.WriteString(`<meta property="` + html.EscapeString(property) + `" content="` + html.EscapeString(content) + `">`)
	}

	writeMetaTag("description", meta.Description)
	writeMetaTag("keywords", meta.Keywords)
	if strings.TrimSpace(meta.Robots) != "" {
		writeMetaTag("robots", meta.Robots)
	}
	if strings.TrimSpace(meta.Canonical) != "" {
		head.WriteString(`<link rel="canonical" href="` + html.EscapeString(meta.Canonical) + `">`)
	}

	ogType := strings.TrimSpace(meta.Type)
	if ogType == "" {
		ogType = "website"
	}
	writeProperty("og:type", ogType)
	writeProperty("og:title", meta.Title)
	writeProperty("og:description", meta.Description)
	writeProperty("og:url", meta.Canonical)
	writeProperty("og:image", meta.Image)
	writeProperty("og:locale", locale)
	head.WriteString(`<meta name="twitter:card" content="summary_large_image">`)
	writeMetaTag("twitter:title", meta.Title)
	writeMetaTag("twitter:description", meta.Description)
	writeMetaTag("twitter:image", meta.Image)

	for lang, href := range meta.Alternates {
		if strings.TrimSpace(lang) == "" || strings.TrimSpace(href) == "" {
			continue
		}
		head.WriteString(`<link rel="alternate" hreflang="` + html.EscapeString(lang) + `" href="` + html.EscapeString(href) + `">`)
	}

	if strings.TrimSpace(meta.JSONLD) != "" {
		safe := strings.ReplaceAll(meta.JSONLD, "</", `<\/`)
		pathAttr := ""
		if strings.TrimSpace(meta.Path) != "" {
			pathAttr = ` data-path="` + html.EscapeString(meta.Path) + `"`
		}
		localeAttr := ` data-locale="` + html.EscapeString(locale) + `"`
		head.WriteString(`<script type="application/ld+json" data-seo-jsonld="server"` + pathAttr + localeAttr + `>` + safe + `</script>`)
	}

	if head.Len() > 0 {
		insert := []byte(head.String())
		if bytes.Contains(result, []byte("</head>")) {
			result = bytes.Replace(result, []byte("</head>"), append(insert, []byte("</head>")...), 1)
		} else {
			result = append(result, insert...)
		}
	}

	// 正文摘要直接写进 SPA 挂载点：爬虫不执行 JS 也能读到，Vue 挂载时整体替换。
	if strings.TrimSpace(meta.BodyHTML) != "" {
		// 首屏骨架必须保留给浏览器首次绘制；正文快照插到骨架之后，
		// 这样首屏先看到骨架，爬虫仍能读到正文，客户端挂载时再由 Vue 整体替换。
		endMarker := []byte("<!--boot-skeleton-end-->")
		end := bytes.Index(result, endMarker)
		if end >= 0 {
			end += len(endMarker)
			result = append(append(append([]byte{}, result[:end]...), []byte(meta.BodyHTML)...), result[end:]...)
		} else {
			appRoot := []byte(`<div id="app"></div>`)
			body := append([]byte(`<div id="app">`), []byte(meta.BodyHTML)...)
			body = append(body, []byte(`</div>`)...)
			if bytes.Contains(result, appRoot) {
				result = bytes.Replace(result, appRoot, body, 1)
			}
		}
	}

	return result
}

// serveUserIndex 在普通入口响应上叠加服务端 SEO 元数据。
func serveUserIndex(w http.ResponseWriter, body []byte, meta *PageMeta) {
	if meta != nil {
		body = injectMetaHTML(body, meta)
	}
	w.Header().Set("Cache-Control", indexCacheControl)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
}
