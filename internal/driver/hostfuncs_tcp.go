//go:build !no_extism

package driver

import (
	"context"
	"encoding/binary"
	"io"
	"time"

	extism "github.com/extism/go-sdk"
	"github.com/gonglijing/xunjiFsu/internal/logger"
)

func (m *DriverManager) createTCPHostFunctions(resourceID int64) []extism.HostFunction {
	executor := m.executor
	if executor == nil {
		return nil
	}

	// tcp_transceive: 写后读（与 serial_transceive 对齐）
	tcpTransceive := extism.NewHostFunctionWithStack(
		"tcp_transceive",
		func(ctx context.Context, p *extism.CurrentPlugin, stack []uint64) {
			wPtr := stack[0]
			wSize := int(stack[1])
			rPtr := stack[2]
			rCap := int(stack[3])
			timeoutMs := int(stack[4])

			conn := executor.GetTCPConn(resourceID)
			if conn == nil || wSize <= 0 || rCap <= 0 {
				if conn == nil {
					logger.Warn("TCP connection unavailable", "resource_id", resourceID)
				}
				stack[0] = 0
				return
			}
			if !validI64Ptr(wPtr) || !validI64Ptr(rPtr) {
				logger.Warn("TCP transceive invalid memory pointer", "resource_id", resourceID, "w_ptr", wPtr, "r_ptr", rPtr)
				stack[0] = 0
				return
			}

			req, _ := p.Memory().Read(uint32(wPtr), uint32(wSize))
			if _, err := conn.Write(req); err != nil {
				logger.Warn("TCP write failed", "resource_id", resourceID, "error", err, "req", hexPreview(req, 32))
				executor.UnregisterTCP(resourceID)
				stack[0] = 0
				return
			}

			tout := time.Duration(timeoutMs) * time.Millisecond
			if tout <= 0 {
				tout = executor.tcpReadTimeout()
			}
			_ = conn.SetReadDeadline(time.Now().Add(tout))
			buf := make([]byte, rCap)
			n, err := readModbusTCPResponse(conn, buf)
			if err != nil || n <= 0 {
				if err != nil {
					logger.Warn("TCP read failed", "resource_id", resourceID, "error", err, "req", hexPreview(req, 32))
				} else {
					logger.Warn("TCP read timeout", "resource_id", resourceID, "req", hexPreview(req, 32))
				}
				executor.UnregisterTCP(resourceID)
				stack[0] = 0
				return
			}
			p.Memory().Write(uint32(rPtr), buf[:n])
			stack[0] = uint64(n)
		},
		[]extism.ValueType{
			extism.ValueTypeI64, // wPtr
			extism.ValueTypeI64, // wSize
			extism.ValueTypeI64, // rPtr
			extism.ValueTypeI64, // rCap
			extism.ValueTypeI64, // timeout
		},
		[]extism.ValueType{extism.ValueTypeI64},
	)

	return []extism.HostFunction{tcpTransceive}
}

func readModbusTCPResponse(conn io.Reader, buf []byte) (int, error) {
	if len(buf) < 9 {
		return 0, io.ErrShortBuffer
	}

	if _, err := io.ReadFull(conn, buf[:7]); err != nil {
		return 0, err
	}

	length := int(binary.BigEndian.Uint16(buf[4:6]))
	if length < 2 {
		return 0, io.ErrUnexpectedEOF
	}

	total := 6 + length
	if total > len(buf) {
		return 0, io.ErrShortBuffer
	}

	if _, err := io.ReadFull(conn, buf[7:total]); err != nil {
		return 0, err
	}

	return total, nil
}
