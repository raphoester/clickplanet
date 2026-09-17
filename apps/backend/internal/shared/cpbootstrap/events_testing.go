//go:build testing

package cpbootstrap

import (
	"sync"

	"google.golang.org/protobuf/proto"
)

// RecordedEvents stands in for the bus where a test only needs what was published.
type RecordedEvents struct {
	mu     sync.Mutex
	events []proto.Message
}

func NewRecordedEvents() *RecordedEvents {
	return &RecordedEvents{}
}

func (r *RecordedEvents) Publish(event proto.Message) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.events = append(r.events, event)
}

// Published is every event, in the order it was published.
func (r *RecordedEvents) Published() []proto.Message {
	r.mu.Lock()
	defer r.mu.Unlock()

	return append([]proto.Message(nil), r.events...)
}
