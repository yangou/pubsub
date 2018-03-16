package pubsub

import (
	"github.com/go-redis/redis"
)

var subscriber *Subscriber

func Start(client *redis.Client, options ...Option) {
	subscriber = NewSubscriber(client, options...)
	subscriber.Start()
}

func Stop() { subscriber.Stop() }

func Subscribe(pattern string) (*Subscription, error) {
	return subscriber.Subscribe(pattern)
}

func Unsubscribe(sp *Subscription) {
	subscriber.Unsubscribe(sp)
}

func Publish(topic string, message []byte) error {
	return subscriber.Publish(topic, message)
}
