package web

import (
	"strings"
	"testing"
)

func TestInjectMetaHTML(t *testing.T) {
	index := []byte(`<!doctype html><html lang="en"><head><title></title></head><body><div id="app"></div></body></html>`)
	meta := &PageMeta{
		Title:       `<Dujiao> "Local"`,
		Description: `A "quoted" description`,
		Keywords:    `ip, proxy`,
		Canonical:   `https://example.com/products/demo`,
		Image:       `https://example.com/a.png`,
		Type:        "product",
		Locale:      "zh-CN",
		JSONLD:      `{"@type":"Product","name":"</script><script>alert(1)</script>"}`,
		Alternates: map[string]string{
			"zh-CN":     "https://example.com/products/demo?lang=zh-CN",
			"x-default": "https://example.com/products/demo",
		},
	}

	body := string(injectMetaHTML(index, meta))

	if !strings.Contains(body, `<html lang="zh-CN">`) {
		t.Fatalf("html lang was not replaced: %s", body)
	}
	if !strings.Contains(body, `<title>&lt;Dujiao&gt; &#34;Local&#34;</title>`) {
		t.Fatalf("title not escaped: %s", body)
	}
	if !strings.Contains(body, `content="A &#34;quoted&#34; description"`) {
		t.Fatalf("description not escaped: %s", body)
	}
	if !strings.Contains(body, `rel="canonical" href="https://example.com/products/demo"`) {
		t.Fatalf("canonical missing: %s", body)
	}
	if !strings.Contains(body, `hreflang="zh-CN"`) || !strings.Contains(body, `hreflang="x-default"`) {
		t.Fatalf("alternates missing: %s", body)
	}
	if strings.Contains(body, `</script><script>alert(1)</script>`) {
		t.Fatalf("JSON-LD closing script was not escaped: %s", body)
	}
	if !strings.Contains(body, `<script type="application/ld+json" data-seo-jsonld="server"`) {
		t.Fatalf("JSON-LD missing: %s", body)
	}
}

func TestInjectMetaHTMLWithoutTitleIsNoop(t *testing.T) {
	index := []byte(`<html><head><title></title></head></html>`)
	if body := string(injectMetaHTML(index, &PageMeta{})); body != string(index) {
		t.Fatalf("empty meta should not change index: %s", body)
	}
}

func TestInjectMetaHTMLKeepsBootSkeletonForFirstPaint(t *testing.T) {
	index := []byte(`<html><head></head><body><div id="app"><!--boot-skeleton-start--><div class="boot-skeleton">Dujiao-Next</div><!--boot-skeleton-end--></div></body></html>`)
	meta := &PageMeta{
		Title:    "IEPL",
		BodyHTML: `<article><h1>IEPL 专线</h1></article>`,
	}
	body := string(injectMetaHTML(index, meta))
	if !strings.Contains(body, `<div id="app"><!--boot-skeleton-start--><div class="boot-skeleton">Dujiao-Next</div><!--boot-skeleton-end--><article><h1>IEPL 专线</h1></article></div>`) {
		t.Fatalf("body snapshot should be inserted after boot skeleton: %s", body)
	}
	if !strings.Contains(body, "boot-skeleton") {
		t.Fatalf("boot skeleton must remain for browser first paint: %s", body)
	}
}

func TestInjectMetaHTMLWritesRobotsNoindex(t *testing.T) {
	index := []byte(`<html><head><title></title></head><body><div id="app"></div></body></html>`)
	meta := &PageMeta{
		Title:  "Not Found",
		Robots: "noindex, nofollow",
	}
	body := string(injectMetaHTML(index, meta))
	if !strings.Contains(body, `<meta name="robots" content="noindex, nofollow">`) {
		t.Fatalf("robots noindex missing: %s", body)
	}
}

func TestInjectMetaHTMLReplacesBootBrand(t *testing.T) {
	index := []byte(`<html><head><title></title></head><body><div class="boot-skeleton__brand"><!--boot-brand-start-->Dujiao-Next<!--boot-brand-end--></div></body></html>`)
	meta := &PageMeta{
		Title: "首页",
		Brand: "霓途 Neaixt",
	}

	body := string(injectMetaHTML(index, meta))
	if !strings.Contains(body, `<!--boot-brand-start-->霓途 Neaixt<!--boot-brand-end-->`) {
		t.Fatalf("boot brand not injected: %s", body)
	}
	if strings.Contains(body, "Dujiao-Next") {
		t.Fatalf("default product brand must not remain: %s", body)
	}
}

func TestInjectMetaHTMLEmptyBrandClearsBootBrand(t *testing.T) {
	index := []byte(`<html><head><title></title></head><body><div class="boot-skeleton__brand"><!--boot-brand-start-->Dujiao-Next<!--boot-brand-end--></div></body></html>`)
	meta := &PageMeta{Title: "首页"}

	body := string(injectMetaHTML(index, meta))
	if !strings.Contains(body, `<!--boot-brand-start--><!--boot-brand-end-->`) {
		t.Fatalf("boot brand should be empty: %s", body)
	}
	if strings.Contains(body, "Dujiao-Next") {
		t.Fatalf("empty brand must not fall back to product name: %s", body)
	}
}

func TestInjectMetaHTMLEscapesBootBrand(t *testing.T) {
	index := []byte(`<html><head><title></title></head><body><div class="boot-skeleton__brand"><!--boot-brand-start-->Dujiao-Next<!--boot-brand-end--></div></body></html>`)
	meta := &PageMeta{Title: "首页", Brand: `霓途 & Neaixt <Studio>`}

	body := string(injectMetaHTML(index, meta))
	if !strings.Contains(body, `霓途 &amp; Neaixt &lt;Studio&gt;`) {
		t.Fatalf("boot brand was not escaped: %s", body)
	}
}

func TestInjectMetaHTMLKeepsLegacyBootSkeletonWithoutBrandMarkers(t *testing.T) {
	index := []byte(`<html><head><title></title></head><body><div class="boot-skeleton__brand">Dujiao-Next</div></body></html>`)
	meta := &PageMeta{Title: "首页", Brand: "霓途 Neaixt"}

	body := string(injectMetaHTML(index, meta))
	if !strings.Contains(body, `<div class="boot-skeleton__brand">Dujiao-Next</div>`) {
		t.Fatalf("legacy boot skeleton should remain unchanged: %s", body)
	}
}

func TestInjectMetaHTMLWritesBodySnapshot(t *testing.T) {
	index := []byte(`<html><head><title></title></head><body><div id="app"></div></body></html>`)
	meta := &PageMeta{
		Title:    "IEPL",
		Path:     "/lp/iepl",
		JSONLD:   `{"@type":"WebPage"}`,
		BodyHTML: `<article><h1>IEPL 专线</h1></article>`,
	}
	body := string(injectMetaHTML(index, meta))
	if !strings.Contains(body, `<div id="app"><article><h1>IEPL 专线</h1></article></div>`) {
		t.Fatalf("body snapshot not injected: %s", body)
	}
	if !strings.Contains(body, `data-path="/lp/iepl"`) {
		t.Fatalf("server JSON-LD path marker missing: %s", body)
	}
	if !strings.Contains(body, `data-locale="zh-CN"`) {
		t.Fatalf("server JSON-LD locale marker missing: %s", body)
	}
}
