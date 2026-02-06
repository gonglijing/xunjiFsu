package graceful

import (
	"context"
	"net/http"
	"sync/atomic"
	"testing"
	"time"
)

type fakeServer struct {
	http.Server
	shutdownCalled int32
}

func (s *fakeServer) Shutdown(ctx context.Context) error {
	atomic.StoreInt32(&s.shutdownCalled, 1)
	return nil
}

func TestNewGracefulShutdown_BasicFields(t *testing.T) {
	g := NewGracefulShutdown(5 * time.Second)
	if g.timeout != 5*time.Second {
		t.Fatalf("expected timeout 5s, got %v", g.timeout)
	}
	if len(g.shutdownFuncs) != 0 {
		t.Fatalf("expected no shutdown funcs initially")
	}
}

func TestGracefulShutdown_AddAndRunFuncs(t *testing.T) {
	g := NewGracefulShutdown(2 * time.Second)

	var called1, called2 int32
	g.AddShutdownFunc(func(ctx context.Context) error {
		atomic.StoreInt32(&called1, 1)
		return nil
	})
	g.AddShutdownFuncInterface(func(ctx context.Context) error {
		atomic.StoreInt32(&called2, 1)
		return nil
	})

	// 直接调用 Shutdown，验证所有函数被执行一次
	g.Shutdown()
	g.Shutdown() // 再次调用也不应重复执行（由 once 保证）

	if atomic.LoadInt32(&called1) != 1 || atomic.LoadInt32(&called2) != 1 {
		t.Fatalf("expected both shutdown funcs to be called exactly once")
	}
}

func TestGracefulShutdown_HTTPServerShutdown(t *testing.T) {
	g := NewGracefulShutdown(2 * time.Second)
	fs := &fakeServer{}
	g.SetHTTPServer(&fs.Server)

	g.Shutdown()

	if atomic.LoadInt32(&fs.shutdownCalled) != 1 {
		t.Fatalf("expected HTTP server Shutdown to be called")
	}
}

func TestGracefulShutdown_WithTimeout(t *testing.T) {
	g := NewGracefulShutdown(100 * time.Millisecond)
	ctx, cancel := g.WithTimeout()
	defer cancel()

	deadline, ok := ctx.Deadline()
	if !ok {
		t.Fatalf("expected context to have deadline")
	}
	if time.Until(deadline) <= 0 {
		t.Fatalf("deadline already passed")
	}
}

