package collector

import (
	"os"
)

// ReadSystemMemoryMB returns Linux system memory totals in MB.
func ReadSystemMemoryMB() (total, used, available float64) {
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return 0, 0, 0
	}

	total, used, available, ok := parseSystemMemoryMBBytes(data)
	if !ok {
		return 0, 0, 0
	}
	return total, used, available
}

func parseSystemMemoryMBBytes(data []byte) (total, used, available float64, ok bool) {
	var memTotal int64
	var memAvailable int64
	var memFree int64
	var buffers int64
	var cached int64

	for len(data) > 0 {
		line := data
		if idx := indexByte(data, '\n'); idx >= 0 {
			line = data[:idx]
			data = data[idx+1:]
		} else {
			data = nil
		}

		switch {
		case hasLinePrefix(line, "MemTotal:"):
			memTotal = parseMeminfoValue(line[len("MemTotal:"):])
		case hasLinePrefix(line, "MemAvailable:"):
			memAvailable = parseMeminfoValue(line[len("MemAvailable:"):])
		case hasLinePrefix(line, "MemFree:"):
			memFree = parseMeminfoValue(line[len("MemFree:"):])
		case hasLinePrefix(line, "Buffers:"):
			buffers = parseMeminfoValue(line[len("Buffers:"):])
		case hasLinePrefix(line, "Cached:"):
			cached = parseMeminfoValue(line[len("Cached:"):])
		}

		if memTotal > 0 && memAvailable > 0 {
			break
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

func hasLinePrefix(line []byte, prefix string) bool {
	if len(line) < len(prefix) {
		return false
	}
	for i := 0; i < len(prefix); i++ {
		if line[i] != prefix[i] {
			return false
		}
	}
	return true
}

func parseMeminfoValue(line []byte) int64 {
	start := -1
	var value int64
	for i := 0; i < len(line); i++ {
		ch := line[i]
		if ch >= '0' && ch <= '9' {
			if start < 0 {
				start = i
			}
			value = value*10 + int64(ch-'0')
			continue
		}
		if start >= 0 {
			break
		}
	}
	if start < 0 {
		return 0
	}
	return value
}

func indexByte(s []byte, target byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == target {
			return i
		}
	}
	return -1
}
