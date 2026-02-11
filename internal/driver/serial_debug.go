package driver

import (
	"errors"
	"fmt"
	"net"
	"os"
	"strings"
	"time"
)

// SerialConfig 串口配置
// 用于调试工具等轻量场景，避免重复串口打开逻辑。
type SerialConfig struct {
	BaudRate    int
	DataBits    int
	Parity      string
	StopBits    int
	ReadTimeout time.Duration
}

// OpenSerial 使用 native 实现打开串口。
func OpenSerial(path string, cfg SerialConfig) (SerialPort, error) {
	mode := serialOpenMode{
		BaudRate: cfg.BaudRate,
		DataBits: cfg.DataBits,
		Parity:   strings.ToUpper(strings.TrimSpace(cfg.Parity)),
		StopBits: cfg.StopBits,
	}
	if mode.Parity == "" {
		mode.Parity = "N"
	}

	port, err := openSerialPort(path, mode)
	if err != nil {
		return nil, err
	}

	if cfg.ReadTimeout > 0 {
		if setter, ok := port.(interface{ SetReadTimeout(time.Duration) error }); ok {
			_ = setter.SetReadTimeout(cfg.ReadTimeout)
		}
	}

	return port, nil
}

// TransceiveSerial 打开串口后执行一次写后读交互。
func TransceiveSerial(path string, cfg SerialConfig, request []byte, expectLen int) ([]byte, error) {
	if len(request) == 0 {
		return nil, fmt.Errorf("request is empty")
	}
	if expectLen <= 0 {
		expectLen = 256
	}
	if cfg.ReadTimeout <= 0 {
		cfg.ReadTimeout = 800 * time.Millisecond
	}

	port, err := OpenSerial(path, cfg)
	if err != nil {
		return nil, err
	}
	defer port.Close()

	if resetInput, ok := port.(interface{ ResetInputBuffer() error }); ok {
		_ = resetInput.ResetInputBuffer()
	}
	if resetOutput, ok := port.(interface{ ResetOutputBuffer() error }); ok {
		_ = resetOutput.ResetOutputBuffer()
	}

	if _, err := port.Write(request); err != nil {
		return nil, err
	}

	response := make([]byte, expectLen)
	total := 0
	for total < expectLen {
		n, readErr := port.Read(response[total:])
		if n > 0 {
			total += n
			if total >= expectLen {
				break
			}
			continue
		}
		if readErr == nil {
			continue
		}
		if timeoutError(readErr) {
			if total == 0 {
				return nil, fmt.Errorf("serial read timeout")
			}
			break
		}
		return nil, readErr
	}

	return response[:total], nil
}

func timeoutError(err error) bool {
	if err == nil {
		return false
	}
	if ne, ok := err.(net.Error); ok && ne.Timeout() {
		return true
	}
	return errors.Is(err, os.ErrDeadlineExceeded)
}
