package collector

import "encoding/json"
import "time"

// DeviceRuntimeStatus 表示设备在采集器中的运行时状态快照。
type DeviceRuntimeStatus struct {
	DeviceID            int64     `json:"device_id"`
	Registered          bool      `json:"registered"`
	CollectIntervalMs   int64     `json:"collect_interval_ms"`
	StorageIntervalSec  int64     `json:"storage_interval_sec"`
	NextRunAt           time.Time `json:"-"`
	LastRunAt           time.Time `json:"-"`
	LastStoredAt        time.Time `json:"-"`
	ConsecutiveFailures int       `json:"consecutive_failures"`
	LastError           string    `json:"last_error,omitempty"`
	LastErrorKind       string    `json:"last_error_kind,omitempty"`
	LastErrorAt         time.Time `json:"-"`
}

// ListDeviceRuntimeStatus 返回设备采集状态快照（按设备 ID 索引）。
func (c *Collector) ListDeviceRuntimeStatus() map[int64]DeviceRuntimeStatus {
	c.mu.RLock()
	defer c.mu.RUnlock()

	out := make(map[int64]DeviceRuntimeStatus, len(c.tasks))
	for deviceID, task := range c.tasks {
		if task == nil {
			continue
		}
		out[deviceID] = runtimeStatusFromTask(task)
	}
	return out
}

// GetDeviceRuntimeStatus 返回单设备采集状态快照。
func (c *Collector) GetDeviceRuntimeStatus(deviceID int64) (DeviceRuntimeStatus, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	task, ok := c.tasks[deviceID]
	if !ok || task == nil {
		return DeviceRuntimeStatus{}, false
	}
	return runtimeStatusFromTask(task), true
}

func runtimeStatusFromTask(task *collectTask) DeviceRuntimeStatus {
	status := DeviceRuntimeStatus{
		Registered: true,
	}
	if task == nil {
		return status
	}

	if task.device != nil {
		status.DeviceID = task.device.ID
	}
	if task.interval > 0 {
		status.CollectIntervalMs = task.interval.Milliseconds()
	}
	if task.storageInterval > 0 {
		status.StorageIntervalSec = int64(task.storageInterval / time.Second)
	}
	status.NextRunAt = task.nextRun
	status.LastRunAt = task.lastRun
	status.LastStoredAt = task.lastStored
	status.LastErrorAt = task.lastErrorAt
	status.ConsecutiveFailures = task.consecutiveFailures
	status.LastError = task.lastError
	if task.lastErrorKind != collectErrorKindNone {
		status.LastErrorKind = string(task.lastErrorKind)
	}
	return status
}

func (s DeviceRuntimeStatus) MarshalJSON() ([]byte, error) {
	type runtimeStatusJSON struct {
		DeviceID            int64      `json:"device_id"`
		Registered          bool       `json:"registered"`
		CollectIntervalMs   int64      `json:"collect_interval_ms"`
		StorageIntervalSec  int64      `json:"storage_interval_sec"`
		NextRunAt           *time.Time `json:"next_run_at,omitempty"`
		LastRunAt           *time.Time `json:"last_run_at,omitempty"`
		LastStoredAt        *time.Time `json:"last_stored_at,omitempty"`
		ConsecutiveFailures int        `json:"consecutive_failures"`
		LastError           string     `json:"last_error,omitempty"`
		LastErrorKind       string     `json:"last_error_kind,omitempty"`
		LastErrorAt         *time.Time `json:"last_error_at,omitempty"`
	}

	payload := runtimeStatusJSON{
		DeviceID:            s.DeviceID,
		Registered:          s.Registered,
		CollectIntervalMs:   s.CollectIntervalMs,
		StorageIntervalSec:  s.StorageIntervalSec,
		ConsecutiveFailures: s.ConsecutiveFailures,
		LastError:           s.LastError,
		LastErrorKind:       s.LastErrorKind,
	}
	if !s.NextRunAt.IsZero() {
		payload.NextRunAt = &s.NextRunAt
	}
	if !s.LastRunAt.IsZero() {
		payload.LastRunAt = &s.LastRunAt
	}
	if !s.LastStoredAt.IsZero() {
		payload.LastStoredAt = &s.LastStoredAt
	}
	if !s.LastErrorAt.IsZero() {
		payload.LastErrorAt = &s.LastErrorAt
	}
	return json.Marshal(payload)
}
