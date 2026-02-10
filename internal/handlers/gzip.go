package handlers

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"strings"
	"sync"
)

var gzipWriterPool = sync.Pool{
	New: func() interface{} {
		return gzip.NewWriter(io.Discard)
	},
}

// GzipMiddleware Gzip压缩中间件
func GzipMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !acceptGzip(r.Header.Get("Accept-Encoding")) || hasUpgradeConnection(r.Header.Get("Connection")) {
			next.ServeHTTP(w, r)
			return
		}

		if headerContainsToken(w.Header().Get("Content-Encoding"), "gzip") {
			next.ServeHTTP(w, r)
			return
		}

		gz := gzipWriterPool.Get().(*gzip.Writer)
		gz.Reset(w)
		defer func() {
			_ = gz.Close()
			gz.Reset(io.Discard)
			gzipWriterPool.Put(gz)
		}()

		// 包装ResponseWriter
		gzw := &gzipResponseWriter{
			ResponseWriter: w,
			Writer:         gz,
		}
		// 设置响应头
		w.Header().Set("Content-Encoding", "gzip")
		w.Header().Set("Vary", appendVaryToken(w.Header().Get("Vary"), "Accept-Encoding"))
		next.ServeHTTP(gzw, r)
	})
}

// gzipResponseWriter gzip响应写入器
type gzipResponseWriter struct {
	http.ResponseWriter
	Writer *gzip.Writer
}

func (w *gzipResponseWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}

func (w *gzipResponseWriter) WriteString(s string) (int, error) {
	return io.WriteString(w.Writer, s)
}

func (w *gzipResponseWriter) Flush() {
	_ = w.Writer.Flush()
	if flusher, ok := w.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

// GzipResponse 压缩响应数据
func GzipResponse(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	_, err := gz.Write(data)
	if err != nil {
		return nil, err
	}
	gz.Close()
	return buf.Bytes(), nil
}

// GunzipRequest 解压请求数据
func GunzipRequest(body io.Reader) ([]byte, error) {
	gr, err := gzip.NewReader(body)
	if err != nil {
		return nil, err
	}
	defer gr.Close()
	return io.ReadAll(gr)
}

func acceptGzip(acceptEncoding string) bool {
	return headerContainsToken(acceptEncoding, "gzip")
}

func hasUpgradeConnection(connection string) bool {
	return headerContainsToken(connection, "upgrade")
}

func headerContainsToken(value string, token string) bool {
	token = strings.ToLower(strings.TrimSpace(token))
	if token == "" {
		return false
	}
	for _, part := range strings.Split(value, ",") {
		if strings.ToLower(strings.TrimSpace(part)) == token {
			return true
		}
	}
	return false
}

func appendVaryToken(existing string, token string) string {
	token = strings.TrimSpace(token)
	if token == "" {
		return existing
	}
	if existing == "" {
		return token
	}
	for _, part := range strings.Split(existing, ",") {
		if strings.EqualFold(strings.TrimSpace(part), token) {
			return existing
		}
	}
	return existing + ", " + token
}
