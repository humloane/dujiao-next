package middleware

import (
	"compress/gzip"
	"net/http"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
)

// gzipPool 复用 writer，避免每个请求都分配压缩缓冲。
var gzipPool = sync.Pool{
	New: func() any {
		writer, _ := gzip.NewWriterLevel(nil, gzip.BestSpeed)
		return writer
	},
}

// GzipMiddleware 对文本类响应做 gzip 压缩。
//
// 单二进制部署（Docker / systemd 直连）没有 nginx 兜底，SPA 的 JS/CSS/HTML
// 会以未压缩体积传输，移动网络下首屏要慢好几秒。这里只压缩文本类响应，
// 已经带 Content-Encoding 的响应原样透传，并补 Vary 让缓存正确区分。
func GzipMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method == http.MethodHead || !clientAcceptsGzip(c.Request) {
			c.Next()
			return
		}

		writer := &gzipResponseWriter{ResponseWriter: c.Writer}
		c.Writer = writer
		c.Next()
		writer.close()
	}
}

func clientAcceptsGzip(req *http.Request) bool {
	for _, encoding := range strings.Split(req.Header.Get("Accept-Encoding"), ",") {
		name := strings.TrimSpace(strings.SplitN(encoding, ";", 2)[0])
		if strings.EqualFold(name, "gzip") {
			return true
		}
	}
	return false
}

type gzipResponseWriter struct {
	gin.ResponseWriter
	gz      *gzip.Writer
	decided bool
	enabled bool
}

func (w *gzipResponseWriter) WriteHeader(code int) {
	// 204/304 没有响应体，压缩只会带来错误头部。
	if code != http.StatusNoContent && code != http.StatusNotModified {
		w.decide()
	}
	w.ResponseWriter.WriteHeader(code)
}

func (w *gzipResponseWriter) Write(data []byte) (int, error) {
	if !w.decided {
		w.decide()
	}
	if w.enabled {
		return w.gz.Write(data)
	}
	return w.ResponseWriter.Write(data)
}

func (w *gzipResponseWriter) WriteString(value string) (int, error) {
	return w.Write([]byte(value))
}

// decide 在第一次写出前决定是否压缩：只看响应头，不需要缓冲整个响应体。
func (w *gzipResponseWriter) decide() {
	if w.decided {
		return
	}
	w.decided = true

	header := w.ResponseWriter.Header()
	if header.Get("Content-Encoding") != "" {
		return
	}
	if !compressibleContentType(header.Get("Content-Type")) {
		return
	}

	header.Del("Content-Length")
	header.Set("Content-Encoding", "gzip")
	header.Add("Vary", "Accept-Encoding")

	writer := gzipPool.Get().(*gzip.Writer)
	writer.Reset(w.ResponseWriter)
	w.gz = writer
	w.enabled = true
}

func (w *gzipResponseWriter) close() {
	if w.gz == nil {
		return
	}
	_ = w.gz.Close()
	gzipPool.Put(w.gz)
	w.gz = nil
}

func compressibleContentType(contentType string) bool {
	value := strings.ToLower(strings.TrimSpace(strings.SplitN(contentType, ";", 2)[0]))
	switch value {
	case "":
		// 未显式声明类型时按文本处理：gin 的静态资源会先补全类型。
		return true
	case "application/javascript", "application/x-javascript", "text/javascript",
		"application/json", "application/ld+json", "application/xml", "text/xml",
		"image/svg+xml", "application/manifest+json":
		return true
	}
	return strings.HasPrefix(value, "text/")
}
