package adapters

import "time"

type alarmDrainConfig struct {
	flushAlarm      func() error
	alarmQueueEmpty func() bool
	onFlushError    func(error)
	onDrained       func()
	sleepInterval   time.Duration
}

func drainAlarmQueueOnStop(cfg alarmDrainConfig) {
	sleepInterval := cfg.sleepInterval
	if sleepInterval <= 0 {
		sleepInterval = 100 * time.Millisecond
	}

	for {
		if err := cfg.flushAlarm(); err != nil {
			if cfg.onFlushError != nil {
				cfg.onFlushError(err)
			}
			return
		}
		if cfg.alarmQueueEmpty == nil || cfg.alarmQueueEmpty() {
			if cfg.onDrained != nil {
				cfg.onDrained()
			}
			return
		}
		time.Sleep(sleepInterval)
	}
}
