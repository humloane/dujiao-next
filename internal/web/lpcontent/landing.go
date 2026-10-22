// Package lpcontent 提供落地页文案的服务端副本。
//
// landing.json 由 cmd/gen-lp-content 从前台 locale 导出；前台仍是唯一编辑入口，
// landing_test.go 会在两边不一致时失败，避免服务端 SEO 输出和页面文案漂移。
package lpcontent

import (
	_ "embed"
	"encoding/json"
	"sort"
	"strings"
)

//go:embed landing.json
var raw []byte

// FAQItem 是一条常见问题。
type FAQItem struct {
	Q string `json:"q"`
	A string `json:"a"`
}

// Specs 是「规格与交付」七行。
type Specs struct {
	Region   string `json:"region"`
	Type     string `json:"type"`
	Protocol string `json:"protocol"`
	Devices  string `json:"devices"`
	Delivery string `json:"delivery"`
	Billing  string `json:"billing"`
	Support  string `json:"support"`
}

// Page 是单个落地页的文案。
type Page struct {
	Eyebrow       string    `json:"eyebrow"`
	Title         string    `json:"title"`
	Subtitle      string    `json:"subtitle"`
	Keywords      string    `json:"keywords"`
	Trust         []string  `json:"trust"`
	Pains         []string  `json:"pains"`
	Solutions     []string  `json:"solutions"`
	Scenarios     []string  `json:"scenarios"`
	Specs         Specs     `json:"specs"`
	Faq           []FAQItem `json:"faq"`
	FinalTitle    string    `json:"finalTitle"`
	FinalSubtitle string    `json:"finalSubtitle"`
	PricingNote   string    `json:"pricingNote"`
}

// SectionTitles 是各段标题（三语）。
type SectionTitles struct {
	Pains     string `json:"pains"`
	Solutions string `json:"solutions"`
	Process   string `json:"process"`
	Specs     string `json:"specs"`
	Scenarios string `json:"scenarios"`
	Faq       string `json:"faq"`
}

// Labels 是规格表行标题等公共文案。
type Labels struct {
	SpecRegion   string `json:"specRegion"`
	SpecType     string `json:"specType"`
	SpecProtocol string `json:"specProtocol"`
	SpecDevices  string `json:"specDevices"`
	SpecDelivery string `json:"specDelivery"`
	SpecBilling  string `json:"specBilling"`
	SpecSupport  string `json:"specSupport"`
	ViewProducts string `json:"viewProducts"`
}

// Locale 是单个语言的完整落地页文案。
type Locale struct {
	Sections SectionTitles     `json:"sections"`
	Labels   Labels            `json:"labels"`
	CTA      map[string]string `json:"cta"`
	Pages    map[string]Page   `json:"pages"`
}

var (
	content     map[string]Locale
	rawByLocale map[string]json.RawMessage
)

func init() {
	if err := json.Unmarshal(raw, &content); err != nil {
		panic("lpcontent: invalid landing.json: " + err.Error())
	}
	if err := json.Unmarshal(raw, &rawByLocale); err != nil {
		panic("lpcontent: invalid landing.json: " + err.Error())
	}
}

// Lookup 返回指定语言与 slug 的落地页文案；不存在时返回 false。
func Lookup(locale, slug string) (Page, Locale, bool) {
	loc, ok := content[locale]
	if !ok {
		loc, ok = content["zh-CN"]
	}
	if !ok {
		return Page{}, Locale{}, false
	}
	page, ok := loc.Pages[strings.TrimSpace(slug)]
	return page, loc, ok
}

// Slugs 返回全部落地页 slug（排序后），供 sitemap / 测试使用。
func Slugs() []string {
	loc, ok := content["zh-CN"]
	if !ok {
		return nil
	}
	slugs := make([]string, 0, len(loc.Pages))
	for slug := range loc.Pages {
		slugs = append(slugs, slug)
	}
	sort.Strings(slugs)
	return slugs
}

// Locales 返回已内置的语言列表。
func Locales() []string {
	locales := make([]string, 0, len(content))
	for locale := range content {
		locales = append(locales, locale)
	}
	sort.Strings(locales)
	return locales
}
