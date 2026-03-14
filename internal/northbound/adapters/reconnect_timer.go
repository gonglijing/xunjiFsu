package adapters

import "time"

type reconnectScheduler struct {
	timer *time.Timer
	ch    <-chan time.Time
}

func (s *reconnectScheduler) Channel() <-chan time.Time {
	if s == nil {
		return nil
	}
	return s.ch
}

func (s *reconnectScheduler) Schedule(delay time.Duration) {
	if s == nil {
		return
	}
	if s.timer == nil {
		s.timer = time.NewTimer(delay)
		s.ch = s.timer.C
		return
	}
	if !s.timer.Stop() {
		select {
		case <-s.timer.C:
		default:
		}
	}
	s.timer.Reset(delay)
	s.ch = s.timer.C
}

func (s *reconnectScheduler) Stop() {
	if s == nil {
		return
	}
	if s.timer == nil {
		s.ch = nil
		return
	}
	if !s.timer.Stop() {
		select {
		case <-s.timer.C:
		default:
		}
	}
	s.ch = nil
}

func (s *reconnectScheduler) Close() {
	if s == nil {
		return
	}
	s.Stop()
	s.timer = nil
}
