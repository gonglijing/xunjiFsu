package adapters

import "time"

const (
	defaultReportInterval    = 5 * time.Second
	defaultAlarmInterval     = 2 * time.Second
	defaultReconnectInterval = 5 * time.Second
	minUploadInterval        = 500 * time.Millisecond
	defaultAlarmBatch        = 20
	defaultAlarmQueue        = 1000
	defaultRealtimeQueue     = 1000
)
