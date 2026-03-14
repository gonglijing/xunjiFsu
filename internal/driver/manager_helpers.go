//go:build !no_extism

package driver

import (
	"context"
	"errors"
	"fmt"
	"time"
)

func validI64Ptr(ptr uint64) bool {
	return ptr != 0 && ptr <= uint64(^uint32(0))
}

func validI64PtrSize(ptr uint64, size int) bool {
	return validI64Ptr(ptr) && size > 0
}

func readWithTimeout(port SerialPort, buf []byte, expect int, timeout time.Duration) (int, error) {
	deadline := time.Now().Add(timeout)
	read := 0
	for read < expect && time.Now().Before(deadline) {
		n, err := port.Read(buf[read:expect])
		if n > 0 {
			read += n
		}
		if err != nil {
			return read, err
		}
		if read >= expect {
			break
		}
		time.Sleep(2 * time.Millisecond)
	}
	if read < expect {
		return read, fmt.Errorf("timeout")
	}
	return read, nil
}

var ErrPluginEmptyOutput = errors.New("plugin returned empty output")

func callPlugin(ctx context.Context, driver *WasmDriver, function string, input []byte) (uint32, []byte, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	rc, output, err := driver.plugin.CallWithContext(ctx, function, input)
	if err != nil {
		return rc, nil, err
	}

	if len(output) == 0 {
		if alt, err2 := driver.plugin.GetOutput(); err2 == nil && len(alt) > 0 {
			output = alt
		}
	}

	if len(output) == 0 {
		errMsg := driver.plugin.GetError()
		if errMsg != "" {
			return rc, nil, fmt.Errorf("%w: %s", ErrPluginEmptyOutput, errMsg)
		}
		return rc, nil, ErrPluginEmptyOutput
	}

	return rc, output, nil
}
