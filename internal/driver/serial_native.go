package driver

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
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
	if err := configureSerialPort(path, mode); err != nil {
		return nil, err
	}
	file, err := os.OpenFile(path, os.O_RDWR, 0)
	if err != nil {
		return nil, err
	}

	port := &nativeSerialPort{file: file}
	_ = mode
	return port, nil
}

func configureSerialPort(path string, mode serialOpenMode) error {
	args, err := buildSttyArgs(path, mode)
	if err != nil {
		return err
	}

	cmd := exec.Command("stty", args...)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("configure serial %s failed: %w (%s)", path, err, string(output))
	}

	return nil
}

func buildSttyArgs(path string, mode serialOpenMode) ([]string, error) {
	if path == "" {
		return nil, fmt.Errorf("serial path is empty")
	}
	if mode.BaudRate <= 0 {
		mode.BaudRate = 9600
	}
	if mode.DataBits < 5 || mode.DataBits > 8 {
		mode.DataBits = 8
	}
	if mode.StopBits != 2 {
		mode.StopBits = 1
	}

	pathFlag := "-F"
	if runtime.GOOS == "darwin" {
		pathFlag = "-f"
	}

	args := []string{pathFlag, path, strconv.Itoa(mode.BaudRate)}
	args = append(args, fmt.Sprintf("cs%d", mode.DataBits))

	if mode.StopBits == 2 {
		args = append(args, "cstopb")
	} else {
		args = append(args, "-cstopb")
	}

	switch mode.Parity {
	case "E", "EVEN":
		args = append(args, "parenb", "-parodd")
	case "O", "ODD":
		args = append(args, "parenb", "parodd")
	default:
		args = append(args, "-parenb")
	}

	// 关闭规范模式与回显，原始读写
	args = append(args, "raw", "-echo")
	return args, nil
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
