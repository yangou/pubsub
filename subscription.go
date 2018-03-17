package pubsub

import (
	"sync"
	"time"
)

type Message struct {
	Topic   string
	Pattern string
	Matches []string
	Payload []byte
}

type Subscription struct {
	id        string
	pattern   string
	messages  chan *Message
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

func (s *Subscription) cast(topic string, matches []string, payload []byte, timeout time.Duration, logger Logger) {
	s.mux.RLock()
	defer s.mux.RUnlock()

	if s.closed {
		logger.Warn("pubsub: subscription closed")
	} else {
		select {
		case <-time.After(timeout):
			logger.Crit("pubsub: timeout sending message to subscription on pattern %s\n%s", s.pattern, payload)
		case s.messages <- &Message{
			Topic:   topic,
			Pattern: s.pattern,
			Matches: matches,
			Payload: payload,
		}:
		}
	}
}

func (s *Subscription) Messages() <-chan *Message { return s.messages }
