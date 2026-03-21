package collector

import (
	"context"
	"errors"
	"log/slog"
	"strings"
)

type collectErrorKind string

const (
	collectErrorKindNone     collectErrorKind = "none"
	collectErrorKindTimeout  collectErrorKind = "timeout"
	collectErrorKindNetwork  collectErrorKind = "network"
	collectErrorKindResource collectErrorKind = "resource"
	collectErrorKindDriver   collectErrorKind = "driver"
	collectErrorKindCanceled collectErrorKind = "canceled"
	collectErrorKindUnknown  collectErrorKind = "unknown"
)

func classifyCollectError(err error) collectErrorKind {
	if err == nil {
		return collectErrorKindNone
	}
	if errors.Is(err, context.Canceled) {
		return collectErrorKindCanceled
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return collectErrorKindTimeout
	}

	msg := strings.ToLower(strings.TrimSpace(err.Error()))
	switch {
	case strings.Contains(msg, "timeout"), strings.Contains(msg, "deadline exceeded"), strings.Contains(msg, "i/o timeout"):
		return collectErrorKindTimeout
	case strings.Contains(msg, "connection refused"),
		strings.Contains(msg, "connection reset"),
		strings.Contains(msg, "broken pipe"),
		strings.Contains(msg, "network is unreachable"),
		strings.Contains(msg, "no route to host"):
		return collectErrorKindNetwork
	case strings.Contains(msg, "serial"),
		strings.Contains(msg, "tty"),
		strings.Contains(msg, "resource"),
		strings.Contains(msg, "port"):
		return collectErrorKindResource
	case strings.Contains(msg, "driver"),
		strings.Contains(msg, "plugin"),
		strings.Contains(msg, "wasm"):
		return collectErrorKindDriver
	default:
		return collectErrorKindUnknown
	}
}

func (c *Collector) handleCollectFailure(task *collectTask, err error) {
	if task == nil || task.device == nil || err == nil {
		return
	}

	kind := classifyCollectError(err)
	consecutive := c.markTaskFailed(task, err, kind)
	slog.Error("Failed to collect device",
		"name", task.device.Name, "id", task.device.ID,
		"kind", kind, "consecutive_failures", consecutive, "error", err)
}
