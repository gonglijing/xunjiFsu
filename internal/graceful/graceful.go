package graceful

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

// GracefulShutdown 优雅关闭管理器
type GracefulShutdown struct {
	timeout       time.Duration
	shutdownFuncs []func(ctx context.Context) error
	httpServer    httpShutdowner
	notifyChan    chan os.Signal
	once          sync.Once
	wg            sync.WaitGroup
}

type httpShutdowner interface {
	Shutdown(ctx context.Context) error
}

// ShutdownFunc 关闭函数类型
type ShutdownFunc func(ctx context.Context) error

// NewGracefulShutdown 创建优雅关闭管理器
func NewGracefulShutdown(timeout time.Duration) *GracefulShutdown {
	return &GracefulShutdown{
		timeout:       timeout,
		shutdownFuncs: make([]func(ctx context.Context) error, 0),
		notifyChan:    make(chan os.Signal, 1),
	}
}

// AddShutdownFunc 添加关闭函数
func (g *GracefulShutdown) AddShutdownFunc(f ShutdownFunc) {
	g.shutdownFuncs = append(g.shutdownFuncs, f)
}

// SetHTTPServer 设置HTTP服务器
func (g *GracefulShutdown) SetHTTPServer(srv *http.Server) {
	g.httpServer = srv
}

// Start 启动监听
func (g *GracefulShutdown) Start() {
	signal.Notify(g.notifyChan, syscall.SIGINT, syscall.SIGTERM)

	g.wg.Add(1)
	go func() {
		defer g.wg.Done()
		<-g.notifyChan
		slog.Info("Received shutdown signal, starting graceful shutdown")
		g.Shutdown()
	}()
}

// Shutdown 执行关闭
func (g *GracefulShutdown) Shutdown() {
	g.once.Do(func() {
		ctx, cancel := context.WithTimeout(context.Background(), g.timeout)
		defer cancel()

		// 关闭HTTP服务器
		if g.httpServer != nil {
			slog.Info("Shutting down HTTP server")
			if err := g.httpServer.Shutdown(ctx); err != nil {
				slog.Error("HTTP server shutdown error", "error", err)
			}
		}

		// 执行注册的关闭函数
		for i, f := range g.shutdownFuncs {
			slog.Info("Executing shutdown function", "step", i+1, "total", len(g.shutdownFuncs))
			if err := f(ctx); err != nil {
				slog.Error("Shutdown function error", "step", i+1, "error", err)
			}
		}

		slog.Info("Graceful shutdown completed")
	})
}

// Wait 等待关闭完成
func (g *GracefulShutdown) Wait() {
	g.wg.Wait()
}
