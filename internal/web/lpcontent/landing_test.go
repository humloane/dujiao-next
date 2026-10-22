package lpcontent

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// TestEmbeddedCopyMatchesFrontendLocales 防止服务端 SEO 文案与前台 locale 漂移。
// 前台 locale 是唯一事实来源；改完文案后运行 go generate ./internal/web/lpcontent
// 或 go run ./cmd/gen-lp-content 重新导出。
func TestEmbeddedCopyMatchesFrontendLocales(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	for _, locale := range Locales() {
		path := filepath.Join(root, "frontend", "user", "src", "i18n", "locales", locale+".json")
		rawLocale, err := os.ReadFile(path)
		if err != nil {
			t.Skipf("frontend locales not available: %v", err)
		}
		var doc struct {
			Landing json.RawMessage `json:"landing"`
		}
		if err := json.Unmarshal(rawLocale, &doc); err != nil {
			t.Fatalf("%s: %v", path, err)
		}

		var want, got map[string]interface{}
		if err := json.Unmarshal(doc.Landing, &want); err != nil {
			t.Fatalf("%s: %v", path, err)
		}
		if err := json.Unmarshal(loaded()[locale], &got); err != nil {
			t.Fatalf("decode embedded %s: %v", locale, err)
		}

		if !reflect.DeepEqual(want, got) {
			t.Fatalf("%s: embedded landing.json is stale, run go run ./cmd/gen-lp-content", path)
		}
	}
}

func TestLookupFallsBackToSimplifiedChinese(t *testing.T) {
	page, _, ok := Lookup("fr-FR", "iepl")
	if !ok || page.Title == "" {
		t.Fatalf("expected fallback content, got ok=%v title=%q", ok, page.Title)
	}
	if _, _, ok := Lookup("zh-CN", "not-a-slug"); ok {
		t.Fatal("unknown slug must not resolve")
	}
}

func TestSlugsMatchExpectedSet(t *testing.T) {
	slugs := Slugs()
	want := []string{"asia-ip", "enterprise", "iepl", "tiktok-live-ip", "us-residential-ip"}
	if !reflect.DeepEqual(slugs, want) {
		t.Fatalf("slugs = %v, want %v", slugs, want)
	}
}
