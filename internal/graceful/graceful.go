package graceful

import (
	"context"
	"log"
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
	httpServer    *http.Server
	notifyChan    chan os.Signal
	once          sync.Once
	wg            sync.WaitGroup
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

// AddShutdownFuncInterface 添加关闭函数(接口形式)
func (g *GracefulShutdown) AddShutdownFuncInterface(f func(ctx context.Context) error) {
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
		log.Println("[graceful] Received shutdown signal, starting graceful shutdown...")
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
			log.Println("[graceful] Shutting down HTTP server...")
			if err := g.httpServer.Shutdown(ctx); err != nil {
				log.Printf("[graceful] HTTP server shutdown error: %v", err)
			}
		}

		// 执行注册的关闭函数
		for i, f := range g.shutdownFuncs {
			log.Printf("[graceful] Executing shutdown function %d/%d...", i+1, len(g.shutdownFuncs))
			if err := f(ctx); err != nil {
				log.Printf("[graceful] Shutdown function %d error: %v", i+1, err)
			}
		}

		log.Println("[graceful] Graceful shutdown completed")
	})
}

// Wait 等待关闭完成
func (g *GracefulShutdown) Wait() {
	g.wg.Wait()
}

// WithTimeout 创建带超时的上下文
func (g *GracefulShutdown) WithTimeout() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), g.timeout)
}

// WaitForSignal 等待关闭信号
func (g *GracefulShutdown) WaitForSignal() {
	<-g.notifyChan
}
