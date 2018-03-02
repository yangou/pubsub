package pubsub

import (
	"sync"
	"time"
)

type Subscription struct {
	id        string
	pattern   string
	messages  chan []byte
	closed    bool
	mux       sync.RWMutex
	closeOnce sync.Once
}

func (s *Subscription) close() {
	s.closeOnce.Do(func() {
		s.mux.Lock()
		defer s.mux.Unlock()

		s.closed = true
		close(s.messages)
	})
}

func (s *Subscription) cast(message []byte, timeout time.Duration, logger Logger) {
	s.mux.RLock()
	defer s.mux.RUnlock()

	if s.closed {
		logger.Warn("pubsub: subscription closed")
	} else {
		select {
		case <-time.After(timeout):
			logger.Crit("pubsub: timeout sending message to subscription on pattern %s\n%s", s.pattern, message)
		case s.messages <- message:
		}
	}
}

func (s *Subscription) Messages() <-chan []byte { return s.messages }
