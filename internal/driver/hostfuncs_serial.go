package driver

import (
	"context"
	"time"

	extism "github.com/extism/go-sdk"
	"github.com/gonglijing/xunjiFsu/internal/logger"
)

func (m *DriverManager) createSerialHostFunctions(resourceID int64) []extism.HostFunction {
	executor := m.executor
	if executor == nil {
		return nil
	}

	// serial_read: 从串口读取数据
	serialRead := extism.NewHostFunctionWithStack(
		"serial_read",
		func(ctx context.Context, p *extism.CurrentPlugin, stack []uint64) {
			ptr := stack[0]
			size := int(stack[1]) // 读取请求的大小
			if !validI64PtrSize(ptr, size) {
				stack[0] = 0 // 返回 0 表示失败
				return
			}
			buf := make([]byte, size)

			port := executor.GetSerialPort(resourceID)
			if port == nil {
				stack[0] = 0 // 返回 0 表示失败
				return
			}

			n, err := port.Read(buf)
			if err != nil {
				stack[0] = 0
				return
			}

			// 将数据写入插件内存
			p.Memory().Write(uint32(ptr), buf[:n])
			stack[0] = uint64(n) // 返回实际读取的字节数
		},
		[]extism.ValueType{extism.ValueTypeI64, extism.ValueTypeI64},
		[]extism.ValueType{extism.ValueTypeI64},
	)

	// serial_write: 向串口写入数据
	serialWrite := extism.NewHostFunctionWithStack(
		"serial_write",
		func(ctx context.Context, p *extism.CurrentPlugin, stack []uint64) {
			ptr := stack[0]
			size := int(stack[1])
			if !validI64PtrSize(ptr, size) {
				stack[0] = 0
				return
			}

			// 从插件内存读取数据
			data, _ := p.Memory().Read(uint32(ptr), uint32(size))

			port := executor.GetSerialPort(resourceID)
			if port == nil {
				stack[0] = 0
				return
			}

			n, err := port.Write(data)
			if err != nil {
				stack[0] = 0
				return
			}

			stack[0] = uint64(n) // 返回实际写入的字节数
		},
		[]extism.ValueType{extism.ValueTypeI64, extism.ValueTypeI64},
		[]extism.ValueType{extism.ValueTypeI64},
	)

	// serial_transceive: 先写再读（用于半双工协议，如自定义 RTU）
	serialTransceive := extism.NewHostFunctionWithStack(
		"serial_transceive",
		func(ctx context.Context, p *extism.CurrentPlugin, stack []uint64) {
			writePtr := stack[0]
			writeSize := int(stack[1])
			readPtr := stack[2]
			readCap := int(stack[3])
			timeoutMs := int(stack[4])

			port := executor.GetSerialPort(resourceID)
			if port == nil || writeSize <= 0 || readCap <= 0 {
				if port == nil {
					logger.Warn("Serial port not found", "resource_id", resourceID)
				}
				stack[0] = 0
				return
			}
			if !validI64Ptr(writePtr) || !validI64Ptr(readPtr) {
				stack[0] = 0
				return
			}

			if resetIn, ok := port.(interface{ ResetInputBuffer() error }); ok {
				_ = resetIn.ResetInputBuffer()
			}
			if resetOut, ok := port.(interface{ ResetOutputBuffer() error }); ok {
				_ = resetOut.ResetOutputBuffer()
			}

			if rts, ok := port.(interface{ SetRTS(bool) error }); ok {
				_ = rts.SetRTS(true)
			}
			if dtr, ok := port.(interface{ SetDTR(bool) error }); ok {
				_ = dtr.SetDTR(true)
			}

			req, _ := p.Memory().Read(uint32(writePtr), uint32(writeSize))
			if n, err := port.Write(req); err != nil {
				logger.Warn("Serial write failed", "resource_id", resourceID, "error", err)
				stack[0] = 0
				return
			} else {
				logger.Info("Serial write", "resource_id", resourceID, "written", n, "len", len(req), "req", hexPreview(req, 32))
			}

			if rts, ok := port.(interface{ SetRTS(bool) error }); ok {
				_ = rts.SetRTS(false)
			}
			if dtr, ok := port.(interface{ SetDTR(bool) error }); ok {
				_ = dtr.SetDTR(false)
			}
			time.Sleep(3 * time.Millisecond)

			buf := make([]byte, readCap)
			tout := time.Duration(timeoutMs)
			if tout <= 0 {
				tout = executor.serialReadTimeout()
			}
			n, err := readWithTimeout(port, buf, readCap, tout)
			if n == 0 {
				if err != nil {
					logger.Warn("Serial read failed", "resource_id", resourceID, "error", err.Error(), "req", hexPreview(req, 32))
				} else {
					logger.Warn("Serial read timeout", "resource_id", resourceID, "req", hexPreview(req, 32))
				}
				stack[0] = 0
				return
			}
			if err != nil {
				logger.Warn("Serial read partial", "resource_id", resourceID, "read", n, "error", err.Error())
			}
			p.Memory().Write(uint32(readPtr), buf[:n])
			stack[0] = uint64(n)
		},
		[]extism.ValueType{
			extism.ValueTypeI64, // write ptr
			extism.ValueTypeI64, // write size
			extism.ValueTypeI64, // read ptr
			extism.ValueTypeI64, // read cap
			extism.ValueTypeI64, // timeout ms
		},
		[]extism.ValueType{extism.ValueTypeI64}, // bytes read
	)

	// sleep_ms: 毫秒延时
	sleepMs := extism.NewHostFunctionWithStack(
		"sleep_ms",
		func(ctx context.Context, p *extism.CurrentPlugin, stack []uint64) {
			ms := int(stack[0])
			time.Sleep(time.Duration(ms) * time.Millisecond)
		},
		[]extism.ValueType{extism.ValueTypeI64},
		[]extism.ValueType{},
	)

	return []extism.HostFunction{serialRead, serialWrite, serialTransceive, sleepMs}
}
