package adapters

import (
	"strconv"
	"sync/atomic"
	"time"
)

func nextPrefixedID(prefix string, seq *uint64) string {
	n := atomic.AddUint64(seq, 1)
	millis := strconv.FormatInt(time.Now().UnixMilli(), 10)
	return prefix + "_" + millis + "_" + strconv.FormatUint(n, 10)
}
