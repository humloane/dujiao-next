package middleware

import (
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func newGzipTestRouter(handler gin.HandlerFunc) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(GzipMiddleware())
	router.GET("/text", handler)
	router.GET("/image", func(c *gin.Context) {
		c.Header("Content-Type", "image/png")
		c.String(http.StatusOK, strings.Repeat("binary", 500))
	})
	router.POST("/empty", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})
	return router
}

func TestGzipMiddlewareCompressesTextResponses(t *testing.T) {
	router := newGzipTestRouter(func(c *gin.Context) {
		c.Header("Content-Type", "text/html; charset=utf-8")
		c.String(http.StatusOK, strings.Repeat("<p>landing</p>", 200))
	})

	req := httptest.NewRequest(http.MethodGet, "/text", nil)
	req.Header.Set("Accept-Encoding", "gzip, deflate")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Header().Get("Content-Encoding") != "gzip" {
		t.Fatalf("expected gzip encoding, got %q", rec.Header().Get("Content-Encoding"))
	}
	if rec.Header().Get("Content-Length") != "" {
		t.Fatalf("content-length must be dropped, got %q", rec.Header().Get("Content-Length"))
	}
	if !strings.Contains(rec.Header().Get("Vary"), "Accept-Encoding") {
		t.Fatalf("missing Vary header: %v", rec.Header())
	}
	if rec.Body.Len() >= len(strings.Repeat("<p>landing</p>", 200)) {
		t.Fatalf("body was not compressed: %d bytes", rec.Body.Len())
	}

	reader, err := gzip.NewReader(rec.Body)
	if err != nil {
		t.Fatalf("gzip reader: %v", err)
	}
	defer reader.Close()
	decoded, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("gzip read: %v", err)
	}
	if !strings.Contains(string(decoded), "landing") {
		t.Fatalf("unexpected decoded body: %q", string(decoded))
	}
}

func TestGzipMiddlewareSkipsWhenClientDoesNotAccept(t *testing.T) {
	router := newGzipTestRouter(func(c *gin.Context) {
		c.Header("Content-Type", "application/json")
		c.String(http.StatusOK, `{"status":"ok"}`)
	})

	req := httptest.NewRequest(http.MethodGet, "/text", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Header().Get("Content-Encoding") != "" {
		t.Fatalf("must not compress without Accept-Encoding, got %q", rec.Header().Get("Content-Encoding"))
	}
	if rec.Body.String() != `{"status":"ok"}` {
		t.Fatalf("unexpected body: %q", rec.Body.String())
	}
}

func TestGzipMiddlewareSkipsBinaryContent(t *testing.T) {
	router := newGzipTestRouter(func(c *gin.Context) {
		c.String(http.StatusOK, "unused")
	})

	req := httptest.NewRequest(http.MethodGet, "/image", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Header().Get("Content-Encoding") != "" {
		t.Fatalf("images must not be gzipped, got %q", rec.Header().Get("Content-Encoding"))
	}
}

func TestGzipMiddlewareLeavesEmptyResponsesAlone(t *testing.T) {
	router := newGzipTestRouter(func(c *gin.Context) {
		c.String(http.StatusOK, "unused")
	})

	req := httptest.NewRequest(http.MethodPost, "/empty", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d", rec.Code)
	}
	if rec.Header().Get("Content-Encoding") != "" {
		t.Fatalf("204 must not carry Content-Encoding")
	}
}
