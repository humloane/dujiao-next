package application

import (
	"context"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/dujiao-next/internal/modules/sitemap/contract"
)

type catalogStub struct {
	calls int
}

func (c *catalogStub) ListActiveCategories() ([]contract.Category, error) {
	c.calls++
	return nil, nil
}

func (c *catalogStub) ListActiveProducts(int) ([]contract.Product, error) {
	c.calls++
	return nil, nil
}

type postStub struct {
	calls int
}

func (p *postStub) ListPublishedPosts(context.Context, int) ([]contract.PublishedPost, error) {
	p.calls++
	return nil, nil
}

type cacheStub struct {
	value    string
	setKey   string
	setValue string
	setTTL   time.Duration
}

func (c *cacheStub) GetString(context.Context, string) (string, error) {
	return c.value, nil
}

func (c *cacheStub) SetString(_ context.Context, key, value string, ttl time.Duration) error {
	c.setKey = key
	c.setValue = value
	c.setTTL = ttl
	return nil
}

func TestGenerateReturnsCachedDocumentWithoutReadingSources(t *testing.T) {
	catalog := &catalogStub{}
	posts := &postStub{}
	cache := &cacheStub{value: "<cached/>"}
	service, err := NewService(catalog, posts, cache)
	if err != nil {
		t.Fatalf("new service failed: %v", err)
	}

	got, err := service.Generate(context.Background(), "https://example.com/")
	if err != nil {
		t.Fatalf("generate failed: %v", err)
	}
	if got != "<cached/>" {
		t.Fatalf("cached document got %q", got)
	}
	if catalog.calls != 0 || posts.calls != 0 {
		t.Fatalf("cache hit must skip sources: catalog=%d posts=%d", catalog.calls, posts.calls)
	}
}

func TestGenerateIncludesLandingPagesAndHreflang(t *testing.T) {
	service, err := NewService(&catalogStub{}, &postStub{}, &cacheStub{})
	if err != nil {
		t.Fatalf("new service failed: %v", err)
	}
	document, err := service.Generate(context.Background(), "https://example.com")
	if err != nil {
		t.Fatalf("generate failed: %v", err)
	}
	for _, needle := range []string{
		"https://example.com/lp/us-residential-ip",
		"https://example.com/lp/tiktok-live-ip",
		"https://example.com/lp/iepl",
		"https://example.com/lp/asia-ip",
		"https://example.com/lp/enterprise",
		`xmlns:xhtml="http://www.w3.org/1999/xhtml"`,
		`hreflang="zh-CN"`,
		`hreflang="x-default"`,
	} {
		if !strings.Contains(document, needle) {
			t.Fatalf("sitemap missing %q", needle)
		}
	}
}

func TestGenerateCachesRenderedDocument(t *testing.T) {
	catalog := &catalogStub{}
	posts := &postStub{}
	cache := &cacheStub{}
	service, err := NewService(catalog, posts, cache)
	if err != nil {
		t.Fatalf("new service failed: %v", err)
	}

	document, err := service.Generate(context.Background(), "https://example.com/")
	if err != nil {
		t.Fatalf("generate failed: %v", err)
	}
	if cache.setKey != "sitemap:xml:https://example.com" {
		t.Fatalf("cache key got %q", cache.setKey)
	}
	if cache.setValue != document || cache.setValue == "" {
		t.Fatal("rendered document was not written to cache")
	}
	if cache.setTTL != 5*time.Minute {
		t.Fatalf("cache ttl got %s", cache.setTTL)
	}
	if catalog.calls != 2 || posts.calls != 1 {
		t.Fatalf("cache miss source calls: catalog=%d posts=%d", catalog.calls, posts.calls)
	}
}

func TestRobotsDisallowPathsIncludesConfiguredAdminPath(t *testing.T) {
	cases := []struct {
		admin string
		want  []string
	}{
		{"", []string{"/admin/"}},
		{"/admin", []string{"/admin/"}},
		{"/admin/", []string{"/admin/"}},
		{"/ythumoane", []string{"/admin/", "/ythumoane/"}},
		{"ythumoane/", []string{"/admin/", "/ythumoane/"}},
	}
	for _, tc := range cases {
		got := robotsDisallowPaths(tc.admin)
		if !reflect.DeepEqual(got, tc.want) {
			t.Fatalf("adminPath %q: got %v want %v", tc.admin, got, tc.want)
		}
	}
}
