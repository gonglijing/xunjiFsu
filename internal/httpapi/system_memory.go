package httpapi

import (
	"os"
	"runtime"
	"strconv"
	"strings"

	"github.com/gonglijing/xunjiFsu/internal/collector"
)

func readProcessRSSMB() float64 {
	status, err := os.ReadFile("/proc/self/status")
	if err == nil {
		if kb, ok := readProcStatusKB(string(status), "VmRSS"); ok {
			return float64(kb) / 1024
		}
	}

	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return float64(m.Sys) / 1024 / 1024
}

func readSystemMemoryMB() (total, used, available float64) {
	return collector.ReadSystemMemoryMB()
}

func readProcStatusKB(data, key string) (int64, bool) {
	prefix := key + ":"
	for _, line := range strings.Split(data, "\n") {
		if !strings.HasPrefix(line, prefix) {
			continue
		}
		fields := strings.Fields(strings.TrimSpace(strings.TrimPrefix(line, prefix)))
		if len(fields) == 0 {
			return 0, false
		}
		value, err := strconv.ParseInt(fields[0], 10, 64)
		if err != nil {
			return 0, false
		}
		return value, true
	}
	return 0, false
}
