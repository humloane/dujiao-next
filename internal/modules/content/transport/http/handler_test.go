package contenthttp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	contentapp "github.com/dujiao-next/internal/modules/content/application"
	contentcontract "github.com/dujiao-next/internal/modules/content/contract"
	contentdomain "github.com/dujiao-next/internal/modules/content/domain"
	"github.com/dujiao-next/internal/shared/jsonmap"
	"github.com/gin-gonic/gin"
)

type requestContextKey struct{}

func TestPublicHandlerPassesRequestContextToUseCase(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	posts := &publicPostQueriesStub{}
	handler := NewPublicHandler(posts, nil, nil)
	router := gin.New()
	router.GET("/posts", handler.GetPosts)

	requestContext, cancel := context.WithCancel(context.WithValue(context.Background(), requestContextKey{}, "request-value"))
	cancel()
	request := httptest.NewRequest(http.MethodGet, "/posts?type=blog&page=2&page_size=5", nil).WithContext(requestContext)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if posts.receivedContext == nil {
		t.Fatal("use case did not receive a context")
	}
	if got := posts.receivedContext.Value(requestContextKey{}); got != "request-value" {
		t.Fatalf("context value = %v, want request-value", got)
	}
	if posts.receivedContext.Err() != context.Canceled {
		t.Fatalf("context error = %v, want context.Canceled", posts.receivedContext.Err())
	}
	if posts.receivedQuery.Type != "blog" || posts.receivedQuery.Page != 2 || posts.receivedQuery.PageSize != 5 {
		t.Fatalf("query mapping mismatch: %#v", posts.receivedQuery)
	}
}

func TestAdminHandlerPassesRequestContextToUseCase(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	media := &adminMediaUseCasesStub{}
	handler := NewAdminHandler(nil, nil, nil, media)
	router := gin.New()
	router.GET("/media", handler.GetAdminMedia)

	requestContext := context.WithValue(context.Background(), requestContextKey{}, "admin-request")
	request := httptest.NewRequest(http.MethodGet, "/media?scene=post&page=3&page_size=15", nil).WithContext(requestContext)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if media.receivedContext == nil || media.receivedContext.Value(requestContextKey{}) != "admin-request" {
		t.Fatalf("admin use case received wrong context: %v", media.receivedContext)
	}
	if media.receivedQuery.Scene != "post" || media.receivedQuery.Page != 3 || media.receivedQuery.PageSize != 15 {
		t.Fatalf("media query mapping mismatch: %#v", media.receivedQuery)
	}
}

type publicPostQueriesStub struct {
	receivedContext context.Context
	receivedQuery   contentapp.PublicPostQuery
	posts           []contentdomain.Post
}

var _ PublicPostQueries = (*publicPostQueriesStub)(nil)

func (s *publicPostQueriesStub) ListPublic(ctx context.Context, query contentapp.PublicPostQuery) ([]contentdomain.Post, int64, error) {
	s.receivedContext = ctx
	s.receivedQuery = query
	return s.posts, int64(len(s.posts)), nil
}

func (s *publicPostQueriesStub) GetPublicBySlug(context.Context, string) (*contentdomain.Post, error) {
	return nil, contentcontract.ErrNotFound
}

func (s *publicPostQueriesStub) ListRelatedProducts(context.Context, uint) ([]contentcontract.RelatedProduct, error) {
	return nil, nil
}

type adminMediaUseCasesStub struct {
	receivedContext context.Context
	receivedQuery   contentapp.MediaListQuery
}

var _ AdminMediaUseCases = (*adminMediaUseCasesStub)(nil)

func (s *adminMediaUseCasesStub) List(ctx context.Context, query contentapp.MediaListQuery) ([]contentdomain.Media, int64, error) {
	s.receivedContext = ctx
	s.receivedQuery = query
	return []contentdomain.Media{}, 0, nil
}

func (s *adminMediaUseCasesStub) Rename(context.Context, uint, string) error { return nil }
func (s *adminMediaUseCasesStub) Delete(context.Context, uint) error         { return nil }
func (s *adminMediaUseCasesStub) BatchDelete(context.Context, []uint) (int, []uint) {
	return 0, []uint{}
}

func TestGetPostsReturnsSeoMetaAndCategoryID(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	categoryID := uint(7)
	posts := &publicPostQueriesStub{posts: []contentdomain.Post{{
		ID:          11,
		Slug:        "seo-post",
		Type:        "blog",
		TitleJSON:   jsonmap.JSON{"zh-CN": "文章标题"},
		SeoMetaJSON: jsonmap.JSON{"title": map[string]interface{}{"zh-CN": "SEO 标题"}},
		CategoryID:  &categoryID,
	}}}
	handler := NewPublicHandler(posts, nil, nil)
	router := gin.New()
	router.GET("/posts", handler.GetPosts)

	request := httptest.NewRequest(http.MethodGet, "/posts?type=blog&category_id=7", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if posts.receivedQuery.CategoryID == nil || *posts.receivedQuery.CategoryID != 7 {
		t.Fatalf("category_id not mapped: %#v", posts.receivedQuery)
	}
	var body struct {
		Data []struct {
			SeoMeta    map[string]interface{} `json:"seo_meta"`
			CategoryID *uint                  `json:"category_id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(body.Data) != 1 {
		t.Fatalf("unexpected data: %s", response.Body.String())
	}
	if body.Data[0].CategoryID == nil || *body.Data[0].CategoryID != 7 {
		t.Fatalf("category_id missing in response: %s", response.Body.String())
	}
	title, _ := body.Data[0].SeoMeta["title"].(map[string]interface{})
	if title["zh-CN"] != "SEO 标题" {
		t.Fatalf("seo_meta missing in response: %s", response.Body.String())
	}
}
