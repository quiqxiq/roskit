package pubsub

import "context"

type SubscriberMessage struct {
	Channel string
	Payload []byte
}

type Subscriber interface {
	Subscribe(ctx context.Context, channels ...string) (<-chan SubscriberMessage, error)
}

type NoopSubscriber struct{}

func (NoopSubscriber) Subscribe(ctx context.Context, channels ...string) (<-chan SubscriberMessage, error) {
	ch := make(chan SubscriberMessage)
	close(ch)
	return ch, nil
}

var _ Subscriber = NoopSubscriber{}
