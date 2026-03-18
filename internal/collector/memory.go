package collector

import (
	"os"
	"strconv"
	"strings"
)

// ReadSystemMemoryMB returns Linux system memory totals in MB.
func ReadSystemMemoryMB() (total, used, available float64) {
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return 0, 0, 0
	}

	total, used, available, ok := parseSystemMemoryMB(string(data))
	if !ok {
		return 0, 0, 0
	}
	return total, used, available
}

func parseSystemMemoryMB(data string) (total, used, available float64, ok bool) {
	var memTotal int64
	var memAvailable int64
	var memFree int64
	var buffers int64
	var cached int64

	for _, line := range strings.Split(data, "\n") {
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}

		valueFields := strings.Fields(strings.TrimSpace(parts[1]))
		if len(valueFields) == 0 {
			continue
		}

		value, err := strconv.ParseInt(valueFields[0], 10, 64)
		if err != nil {
			continue
		}

		switch strings.TrimSpace(parts[0]) {
		case "MemTotal":
			memTotal = value
		case "MemAvailable":
			memAvailable = value
		case "MemFree":
			memFree = value
		case "Buffers":
			buffers = value
		case "Cached":
			cached = value
		}
	}

	if memTotal <= 0 {
		return 0, 0, 0, false
	}

	if memAvailable <= 0 {
		memAvailable = memFree + buffers + cached
	}
	if memAvailable < 0 {
		memAvailable = 0
	}
	if memAvailable > memTotal {
		memAvailable = memTotal
	}

	memUsed := memTotal - memAvailable
	if memUsed < 0 {
		memUsed = 0
	}

	return float64(memTotal) / 1024, float64(memUsed) / 1024, float64(memAvailable) / 1024, true
}
