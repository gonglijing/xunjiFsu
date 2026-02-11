//go:build !no_extism

package driver

import (
	"context"
	"time"

	extism "github.com/extism/go-sdk"
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
				stack[0] = 0
				return
			}
			if !validI64Ptr(wPtr) || !validI64Ptr(rPtr) {
				stack[0] = 0
				return
			}

			req, _ := p.Memory().Read(uint32(wPtr), uint32(wSize))
			if _, err := conn.Write(req); err != nil {
				executor.UnregisterTCP(resourceID)
				stack[0] = 0
				return
			}

			tout := time.Duration(timeoutMs)
			if tout <= 0 {
				tout = executor.tcpReadTimeout()
			}
			_ = conn.SetReadDeadline(time.Now().Add(tout))
			buf := make([]byte, rCap)
			n, err := conn.Read(buf)
			if err != nil || n <= 0 {
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
