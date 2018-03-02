package pubsub

import "time"

type Option func(*Subscriber)

func SetChannel(channel string) Option {
	return Option(func(s *Subscriber) {
		s.channel = channel
	})
}

func SetTimeout(timeout time.Duration) Option {
	return Option(func(s *Subscriber) {
		s.timeout = timeout
	})
}

func SetLogger(logger Logger) Option {
	return Option(func(s *Subscriber) {
		s.logger = logger
	})
}

func EnableLogger() Option {
	return Option(func(s *Subscriber) {
		s.logger = &logger{enabled: true}
	})
}

func DisableLogger() Option {
	return Option(func(s *Subscriber) {
		s.logger = &logger{enabled: false}
	})
}
