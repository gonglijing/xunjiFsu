package adapters

import (
	"log"
	"time"
)

type bufferedFlushLoopConfig struct {
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

func runBufferedFlushLoop(cfg bufferedFlushLoopConfig) {
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
					log.Printf("%s alarm flush failed on close: %v", cfg.logLabel, err)
				},
			})
			return
		case <-reportTicker.C:
			if err := cfg.flushData(); err != nil {
				log.Printf("%s %s flush failed: %v", cfg.logLabel, cfg.reportLabel, err)
			}
		case <-alarmTicker.C:
			if err := cfg.flushAlarm(); err != nil {
				log.Printf("%s alarm flush failed: %v", cfg.logLabel, err)
			}
		case <-cfg.flushNow:
			if err := cfg.flushAlarm(); err != nil {
				log.Printf("%s alarm flush failed: %v", cfg.logLabel, err)
			}
		}
	}
}
