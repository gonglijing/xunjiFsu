package driver

import (
	"io"
	"path/filepath"
	"testing"
	"time"

	"github.com/gonglijing/xunjiFsu/internal/database"
	"github.com/gonglijing/xunjiFsu/internal/models"
)

func BenchmarkNewPreparedExecution_ModbusTCP(b *testing.B) {
	tmpDir := b.TempDir()
	paramPath := filepath.Join(tmpDir, "param.db")
	originalParamDB := database.ParamDB
	b.Cleanup(func() {
		if database.ParamDB != nil {
			_ = database.ParamDB.Close()
		}
		database.ParamDB = originalParamDB
	})
	if err := database.InitParamDBWithPath(paramPath); err != nil {
		b.Fatalf("InitParamDBWithPath failed: %v", err)
	}

	resourceID := int64(99)
	device := &models.Device{
		ID:            1,
		Name:          "dev-1",
		DriverType:    "modbus_tcp",
		ResourceID:    &resourceID,
		IPAddress:     "127.0.0.1",
		PortNum:       502,
		DeviceAddress: "1",
	}

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		prepared := NewPreparedExecution(device)
		if prepared == nil {
			b.Fatal("prepared execution is nil")
		}
	}
}

type benchmarkChunkedSerialPort struct {
	frame []byte
	chunk int
	off   int
}

func (p *benchmarkChunkedSerialPort) Read(dst []byte) (int, error) {
	if p.off >= len(p.frame) {
		p.off = 0
	}
	end := p.off + p.chunk
	if end > len(p.frame) {
		end = len(p.frame)
	}
	n := copy(dst, p.frame[p.off:end])
	p.off += n
	return n, nil
}

func (p *benchmarkChunkedSerialPort) Write([]byte) (int, error) { return 0, io.EOF }
func (p *benchmarkChunkedSerialPort) Close() error              { return nil }

func BenchmarkReadWithTimeout_Chunked256(b *testing.B) {
	port := &benchmarkChunkedSerialPort{
		frame: make([]byte, 256),
		chunk: 32,
	}
	buf := make([]byte, 256)

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		n, err := readWithTimeout(port, buf, len(buf), 50*time.Millisecond)
		if err != nil {
			b.Fatalf("readWithTimeout error: %v", err)
		}
		if n != len(buf) {
			b.Fatalf("readWithTimeout n = %d, want %d", n, len(buf))
		}
	}
}

func BenchmarkWasmDriverHasFunction_Cached(b *testing.B) {
	driver := &WasmDriver{
		exportedSet: map[string]struct{}{
			"handle":  {},
			"version": {},
		},
	}

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if !driver.hasFunction("handle") {
			b.Fatal("expected cached handle export")
		}
	}
}
