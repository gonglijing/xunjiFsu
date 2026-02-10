package driver

import (
	"fmt"
	"os"
	"sync"
	"time"
)

type serialOpenMode struct {
	BaudRate int
	DataBits int
	Parity   string
	StopBits int
}

type nativeSerialPort struct {
	file        *os.File
	mu          sync.RWMutex
	readTimeout time.Duration
}

func openSerialPort(path string, mode serialOpenMode) (SerialPort, error) {
	if path == "" {
		return nil, fmt.Errorf("serial path is empty")
	}
	file, err := os.OpenFile(path, os.O_RDWR, 0)
	if err != nil {
		return nil, err
	}

	port := &nativeSerialPort{file: file}
	_ = mode
	return port, nil
}

func (p *nativeSerialPort) Write(data []byte) (int, error) {
	if p == nil || p.file == nil {
		return 0, fmt.Errorf("serial port is nil")
	}
	return p.file.Write(data)
}

func (p *nativeSerialPort) Read(data []byte) (int, error) {
	if p == nil || p.file == nil {
		return 0, fmt.Errorf("serial port is nil")
	}

	timeout := p.getReadTimeout()
	if timeout > 0 {
		_ = p.file.SetReadDeadline(time.Now().Add(timeout))
	}
	return p.file.Read(data)
}

func (p *nativeSerialPort) Close() error {
	if p == nil || p.file == nil {
		return nil
	}
	return p.file.Close()
}

func (p *nativeSerialPort) SetReadTimeout(timeout time.Duration) error {
	if p == nil {
		return fmt.Errorf("serial port is nil")
	}
	p.mu.Lock()
	p.readTimeout = timeout
	p.mu.Unlock()
	return nil
}

func (p *nativeSerialPort) getReadTimeout() time.Duration {
	if p == nil {
		return 0
	}
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.readTimeout
}
