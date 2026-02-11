package driver

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"strings"
	"time"
)

// TCPConfig 调试场景下的 TCP 通讯参数。
type TCPConfig struct {
	Timeout time.Duration
}

// TransceiveTCP 建立短连接完成一次 Modbus TCP 收发。
func TransceiveTCP(endpoint string, cfg TCPConfig, request []byte) ([]byte, error) {
	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" {
		return nil, fmt.Errorf("tcp endpoint is empty")
	}
	if len(request) == 0 {
		return nil, fmt.Errorf("request is empty")
	}

	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 2 * time.Second
	}

	conn, err := net.DialTimeout("tcp", endpoint, timeout)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	_ = conn.SetDeadline(time.Now().Add(timeout))

	if _, err := conn.Write(request); err != nil {
		return nil, err
	}

	header := make([]byte, 7)
	if _, err := io.ReadFull(conn, header); err != nil {
		return nil, err
	}

	protocolID := binary.BigEndian.Uint16(header[2:4])
	if protocolID != 0 {
		return nil, fmt.Errorf("invalid modbus protocol id: %d", protocolID)
	}

	length := int(binary.BigEndian.Uint16(header[4:6]))
	if length < 2 {
		return nil, fmt.Errorf("invalid modbus length: %d", length)
	}

	pduLen := length - 1
	payload := make([]byte, pduLen)
	if _, err := io.ReadFull(conn, payload); err != nil {
		return nil, err
	}

	response := append(header, payload...)
	return response, nil
}
