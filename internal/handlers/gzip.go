package handlers

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"strings"
)

// GzipMiddleware Gzip压缩中间件
func GzipMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 检查客户端是否支持gzip
		acceptEncoding := r.Header.Get("Accept-Encoding")
		if !strings.Contains(acceptEncoding, "gzip") {
			next.ServeHTTP(w, r)
			return
		}
		// 创建gzip响应写入器
		gz := gzip.NewWriter(w)
		defer gz.Close()
		// 包装ResponseWriter
		gzw := &gzipResponseWriter{
			ResponseWriter: w,
			Writer:         gz,
		}
		// 设置响应头
		w.Header().Set("Content-Encoding", "gzip")
		w.Header().Set("Vary", "Accept-Encoding")
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
	return w.Writer.Write([]byte(s))
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
