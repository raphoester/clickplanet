package facts

import (
	"context"
	"time"

	"google.golang.org/protobuf/proto"
)

type Publisher interface {
	Publish(events ...proto.Message)
}

func NewRelay(log *Log, events Publisher, every time.Duration) *Relay {
	return &Relay{log: log, events: events, every: every}
}

// One sender, in the log's order: each stream gets its facts in the order they were told.
type Relay struct {
	log    *Log
	events Publisher
	every  time.Duration
}

func (r *Relay) Name() string { return "facts-relay" }

func (r *Relay) Run(ctx context.Context) {
	ticker := time.NewTicker(r.every)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			r.send()
		case <-ctx.Done():
			r.send()
			return
		}
	}
}

func (r *Relay) send() {
	if told := r.log.Take(); len(told) > 0 {
		r.events.Publish(told...)
	}
}
