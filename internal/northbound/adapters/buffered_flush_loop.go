package adapters

import (
	"log/slog"
	"time"
)

type periodicFlushLoopConfig struct {
	logLabel        string
	reportLabel     string
	reportInterval  time.Duration
	alarmInterval   time.Duration
	stopChan        <-chan struct{}
	flushNow        <-chan struct{}
	flushData       func() error
	flushAlarm      func() error
	alarmQueueEmpty func() bool
}

func executePeriodicFlushLoop(cfg periodicFlushLoopConfig) {
	reportInterval := cfg.reportInterval
	alarmInterval := cfg.alarmInterval

	if reportInterval <= 0 {
		reportInterval = defaultReportInterval
	}
	if alarmInterval <= 0 {
		alarmInterval = defaultAlarmInterval
	}

	reportTicker := time.NewTicker(reportInterval)
	alarmTicker := time.NewTicker(alarmInterval)
	defer reportTicker.Stop()
	defer alarmTicker.Stop()

	for {
		select {
		case <-cfg.stopChan:
			drainAlarmQueueOnStop(alarmDrainConfig{
				flushAlarm:      cfg.flushAlarm,
				alarmQueueEmpty: cfg.alarmQueueEmpty,
				onFlushError: func(err error) {
					slog.Info("Alarm flush failed on close", "label", cfg.logLabel, "error", err)
				},
			})
			return
		case <-reportTicker.C:
			if err := cfg.flushData(); err != nil {
				slog.Info("Periodic flush failed", "label", cfg.logLabel, "report", cfg.reportLabel, "error", err)
			}
		case <-alarmTicker.C:
			if err := cfg.flushAlarm(); err != nil {
				slog.Info("Alarm flush failed", "label", cfg.logLabel, "error", err)
			}
		case <-cfg.flushNow:
			if err := cfg.flushAlarm(); err != nil {
				slog.Info("Alarm flush failed", "label", cfg.logLabel, "error", err)
			}
		}
	}
}
