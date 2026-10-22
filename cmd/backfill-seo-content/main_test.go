package main

import (
	"strings"
	"testing"

	productdomain "github.com/dujiao-next/internal/modules/catalog/product/domain"
	contentdomain "github.com/dujiao-next/internal/modules/content/domain"
	"github.com/dujiao-next/internal/shared/jsonmap"
)

func TestProductSEOTitle(t *testing.T) {
	us := &productdomain.Product{Slug: "US-Residential-IEPL", Country: "US", IPType: productdomain.IPTypeResidential, LineType: productdomain.LineTypeIEPL}
	if got := productSEOTitle(us, "zh-CN", "霓途 Neaixt"); got != "美国静态住宅 IP｜IEPL｜霓途 Neaixt" {
		t.Fatalf("unexpected zh title: %q", got)
	}
	if got := productSEOTitle(us, "en-US", "Neaixt"); got != "United States Static Residential IP | IEPL | Neaixt" {
		t.Fatalf("unexpected en title: %q", got)
	}
	shared := &productdomain.Product{Slug: "shared-ip-260g-unlimited", IPType: productdomain.IPTypeShared}
	if got := productSEOTitle(shared, "zh-CN", "霓途 Neaixt"); got != "260GB 不限时流量包｜共享 IP｜霓途 Neaixt" {
		t.Fatalf("unexpected shared title: %q", got)
	}
}

func TestProductSEODescriptionUsesLocaleOnly(t *testing.T) {
	product := &productdomain.Product{
		DescriptionJSON: jsonmap.JSON{"zh-CN": "简体描述"},
	}
	if got := productSEODescription(product, "标题", "zh-CN"); got != "简体描述" {
		t.Fatalf("unexpected zh description: %q", got)
	}
	got := productSEODescription(product, "标题", "en-US")
	if strings.Contains(got, "简体描述") {
		t.Fatalf("en description must not reuse simplified copy: %q", got)
	}
	if !strings.Contains(got, "See pricing") {
		t.Fatalf("unexpected en description: %q", got)
	}
}

func TestCategoryDescription(t *testing.T) {
	got := categoryDescription("美国住宅 IP", 12, "zh-CN")
	if !strings.Contains(got, "美国住宅 IP") || !strings.Contains(got, "12") {
		t.Fatalf("unexpected category description: %q", got)
	}
}

func TestCleanNoticeText(t *testing.T) {
	input := `<ul><li>共享节点套餐</li><li>静态住宅 IP</li></ul><p>覆盖六个地区。</p>`
	got := cleanNoticeText(input, "zh-CN")
	if strings.Contains(got, "共享节点套餐") {
		t.Fatalf("retired service should be removed: %q", got)
	}
	if !strings.Contains(got, "静态住宅 IP") {
		t.Fatalf("valid service should stay: %q", got)
	}
	if !strings.Contains(got, "全球 16 个地区") {
		t.Fatalf("region phrase should be replaced: %q", got)
	}
}

func TestPickPostCategory(t *testing.T) {
	cases := []struct {
		title string
		want  string
	}{
		{"TikTok 直播网络怎么选", "live-ops"},
		{"跨境电商店铺的 IP 规划", "cross-border"},
		{"客户端配置教程", "tutorials"},
		{"住宅 IP 的 ASN 怎么看", "network-basics"},
	}
	for _, tc := range cases {
		post := &contentdomain.Post{TitleJSON: jsonmap.JSON{"zh-CN": tc.title}}
		if got := pickPostCategory(post); got != tc.want {
			t.Fatalf("title %q: got %q want %q", tc.title, got, tc.want)
		}
	}
}

func TestReplaceRegionPhrases(t *testing.T) {
	got := replaceRegionPhrases("覆盖六个地区", "site_config.brand.site_description.zh-CN")
	if !strings.Contains(got, "全球 16 个地区") {
		t.Fatalf("unexpected replacement: %q", got)
	}
}
