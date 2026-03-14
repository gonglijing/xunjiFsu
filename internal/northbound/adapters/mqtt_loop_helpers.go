package adapters

import (
	"log"
	"time"
)

type mqttLikeLoopConfig struct {
	logLabel        string
	adapterName     string
	interval        time.Duration
	stopChan        <-chan struct{}
	dataChan        <-chan struct{}
	reconnectNow    <-chan struct{}
	flushData       func()
	flushAlarm      func()
	shouldReconnect func() bool
	reconnectOnce   func() error
	reconnectDelay  func() time.Duration
}

func runMQTTLikeLoop(cfg mqttLikeLoopConfig) {
	interval := cfg.interval
	if interval < minUploadInterval {
		interval = minUploadInterval
	}

	dataTicker := time.NewTicker(interval)
	alarmTicker := time.NewTicker(defaultAlarmInterval)
	defer dataTicker.Stop()
	defer alarmTicker.Stop()

	var reconnect reconnectScheduler
	defer reconnect.Close()

	for {
		select {
		case <-cfg.stopChan:
			cfg.flushData()
			cfg.flushAlarm()
			reconnect.Stop()
			return
		case <-cfg.dataChan:
			cfg.flushData()
			cfg.flushAlarm()
		case <-dataTicker.C:
			cfg.flushData()
		case <-alarmTicker.C:
			cfg.flushAlarm()
		case <-cfg.reconnectNow:
			reconnect.Schedule(0)
		case <-reconnect.Channel():
			if !cfg.shouldReconnect() {
				reconnect.Stop()
				continue
			}
			if err := cfg.reconnectOnce(); err != nil {
				delay := cfg.reconnectDelay()
				log.Printf("%s [%s] reconnect failed: %v, retry in %v", cfg.logLabel, cfg.adapterName, err, delay)
				reconnect.Schedule(delay)
				continue
			}
			reconnect.Stop()
		}
	}
}
