package events

import (
	"context"
)

type Publisher interface {
	Publish(ctx context.Context, exchange string, event *Event, headers Headers) error
	Close() error
}
