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

// flushTrackingWriter 记录底层 Flush 次数，用来验证流式响应确实逐段下发。
type flushTrackingWriter struct {
	*httptest.ResponseRecorder
	flushes int
}

func (w *flushTrackingWriter) Flush() {
	w.flushes++
	w.ResponseRecorder.Flush()
}

func TestGzipMiddlewareFlushesStreamingChunks(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(GzipMiddleware())
	router.GET("/stream", func(c *gin.Context) {
		c.Header("Content-Type", "text/plain; charset=utf-8")
		for _, chunk := range []string{"chunk1", "chunk2", "chunk3"} {
			_, _ = c.Writer.WriteString(chunk)
			c.Writer.Flush()
		}
	})

	req := httptest.NewRequest(http.MethodGet, "/stream", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	recorder := httptest.NewRecorder()
	rec := &flushTrackingWriter{ResponseRecorder: recorder}
	router.ServeHTTP(rec, req)

	if rec.Header().Get("Content-Encoding") != "gzip" {
		t.Fatalf("expected gzip encoding, got %q", rec.Header().Get("Content-Encoding"))
	}
	if rec.flushes < 3 {
		t.Fatalf("expected underlying writer to be flushed for each chunk, got %d", rec.flushes)
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
	if string(decoded) != "chunk1chunk2chunk3" {
		t.Fatalf("unexpected stream content: %q", string(decoded))
	}
}

func TestGzipMiddlewareSkipsEventStream(t *testing.T) {
	router := newGzipTestRouter(func(c *gin.Context) {
		c.Header("Content-Type", "text/event-stream")
		_, _ = c.Writer.WriteString("data: hello\n\n")
		c.Writer.Flush()
	})

	req := httptest.NewRequest(http.MethodGet, "/text", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Header().Get("Content-Encoding") != "" {
		t.Fatalf("event-stream must not be gzipped, got %q", rec.Header().Get("Content-Encoding"))
	}
	if rec.Body.String() != "data: hello\n\n" {
		t.Fatalf("unexpected body: %q", rec.Body.String())
	}
}

func TestGzipMiddlewareSkipsUnknownContentType(t *testing.T) {
	router := newGzipTestRouter(func(c *gin.Context) {
		_, _ = c.Writer.Write([]byte("raw-bytes"))
	})

	req := httptest.NewRequest(http.MethodGet, "/text", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Header().Get("Content-Encoding") != "" {
		t.Fatalf("unknown content type must not be gzipped, got %q", rec.Header().Get("Content-Encoding"))
	}
}

func TestGzipMiddlewareKeepsCSVComplete(t *testing.T) {
	router := newGzipTestRouter(func(c *gin.Context) {
		c.Header("Content-Type", "text/csv; charset=utf-8")
		_, _ = c.Writer.WriteString("id,secret\n1,abc\n2,def\n")
		c.Writer.Flush()
	})

	req := httptest.NewRequest(http.MethodGet, "/text", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Header().Get("Content-Encoding") != "gzip" {
		t.Fatalf("expected gzip encoding, got %q", rec.Header().Get("Content-Encoding"))
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
	if string(decoded) != "id,secret\n1,abc\n2,def\n" {
		t.Fatalf("csv content truncated: %q", string(decoded))
	}
}
