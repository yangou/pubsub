package pubsub

import (
	"errors"
	"github.com/go-redis/redis"
	"github.com/satori/go.uuid"
	"math"
	"path"
	"regexp"
	"strings"
	"sync"
	"time"
)

var PubsubStoppedErr = errors.New("pubsub stopped")

func NewSubscriber(client *redis.Client, options ...Option) *Subscriber {
	s := &Subscriber{
		client:  client,
		channel: "pubsub",
		logger:  &logger{enabled: false},
		timeout: 5 * time.Second,
		stopped: true,
	}
	for _, option := range options {
		option(s)
	}
	return s
}

type Subscriber struct {
	client  *redis.Client
	channel string
	logger  Logger
	timeout time.Duration

	subscriber *redis.PubSub
	messages   <-chan *redis.Message

	patterns        map[string]*regexp.Regexp
	subscriptions   map[string]map[string]*Subscription
	stopped         bool
	listenerStopped chan struct{}
	mux             sync.RWMutex
}

func (s *Subscriber) Start() {
	s.patterns = map[string]*regexp.Regexp{}
	s.subscriptions = map[string]map[string]*Subscription{}
	s.stopped = false
	s.listenerStopped = make(chan struct{})
	s.subscriber = s.client.PSubscribe(path.Join(s.channel, "*"))
	s.messages = s.subscriber.Channel()

	go s.listen()
}

func (s *Subscriber) listen() {
	defer close(s.listenerStopped)

	for {
		if msg, more := <-s.messages; more {
			s.logger.Debug("received pubusb message on channel %s\n%s", msg.Channel, msg.Payload)
			topic := strings.Replace(msg.Channel, s.channel+"/", "", 1)
			go s.cast(topic, []byte(msg.Payload))
		} else {
			return
		}
	}
}

func (s *Subscriber) cast(topic string, payload []byte) {
	s.mux.RLock()
	defer s.mux.RUnlock()

	for rexStr, rex := range s.patterns {
		if matches := rex.FindStringSubmatch(topic); len(matches) > 0 {
			for _, sp := range s.subscriptions[rexStr] {
				go sp.cast(topic, matches, payload, s.timeout, s.logger)
			}
		}
	}
}

func (s *Subscriber) Stop() {
	if err := s.subscriber.Close(); err != nil {
		s.logger.Error("pubsub: error closing subscriber, %s", err.Error())
	}
	<-s.listenerStopped

	s.mux.Lock()
	s.stopped = true
	subscriptions := []*Subscription{}
	for _, sps := range s.subscriptions {
		for _, sp := range sps {
			subscriptions = append(subscriptions, sp)
		}
	}

	s.subscriber = nil
	s.messages = nil
	s.patterns = nil
	s.subscriptions = nil
	s.listenerStopped = nil
	s.mux.Unlock()

	for _, sp := range subscriptions {
		go sp.close()
	}
}

func (s *Subscriber) Subscribe(pattern string) (*Subscription, error) {
	rex, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}

	sp := &Subscription{
		id:       uuid.NewV4().String(),
		pattern:  pattern,
		messages: make(chan *Message, math.MaxInt16),
	}

	s.mux.Lock()
	defer s.mux.Unlock()

	if s.stopped {
		return nil, PubsubStoppedErr
	}

	s.patterns[pattern] = rex
	if sps, found := s.subscriptions[pattern]; found {
		sps[sp.id] = sp
	} else {
		s.subscriptions[pattern] = map[string]*Subscription{sp.id: sp}
	}

	return sp, nil
}

func (s *Subscriber) Unsubscribe(sp *Subscription) {
	s.mux.Lock()
	defer s.mux.Unlock()

	sps, found := s.subscriptions[sp.pattern]
	if found {
		delete(sps, sp.id)
	}
	if len(sps) == 0 {
		delete(s.subscriptions, sp.pattern)
		delete(s.patterns, sp.pattern)
	}

	go sp.close()
}

func (s *Subscriber) Publish(topic string, message []byte) error {
	return s.client.Publish(path.Join(s.channel, topic), string(message)).Err()
}
