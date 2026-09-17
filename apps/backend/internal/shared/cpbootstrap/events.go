package cpbootstrap

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// EventBus is how a module tells the others what happened, in process: the emitter never learns who listens.
//
// Delivery is at most once and not durable. Each subscriber reads its own buffered channel in its own
// goroutine, so no subscriber code runs in the publisher's stack trace, and a full buffer drops the event
// and counts it in events_dropped_total. A flow that cannot lose an event needs a durable path of its own.
type EventBus interface {
	// Publish never blocks. The emitter builds a fresh message per event and never touches it again.
	Publish(event proto.Message)

	register(event protoreflect.FullName, subscriber string, buffer int) (<-chan proto.Message, prometheus.Counter, error)
}

// Handler reads one kind of event, in its subscriber's goroutine. An error is counted in events_failed_total;
// a subscriber that wants it logged wraps its handler in a decorator.
type Handler[T proto.Message] interface {
	Handle(ctx context.Context, event T) error
}

// Subscribe registers a subscriber to events of type T, and answers the runner that delivers them.
//
// Call it while the module is built and add the runner: registering after the runners start is refused, so
// no event is published to nobody at boot. Events published before the runner starts wait in the buffer.
func Subscribe[T proto.Message](bus EventBus, subscriber string, buffer int, handler Handler[T]) (Runner, error) {
	var zero T
	event := zero.ProtoReflect().Descriptor().FullName()

	events, failed, err := bus.register(event, subscriber, buffer)
	if err != nil {
		return nil, err
	}

	return &subscription[T]{name: subscriber, events: events, handler: failures[T]{inner: handler, failed: failed}}, nil
}

type subscription[T proto.Message] struct {
	name    string
	events  <-chan proto.Message
	handler Handler[T]
}

func (s *subscription[T]) Name() string { return "events:" + s.name }

// Run delivers until ctx is done, then delivers what is already buffered, so a subscriber stopped in a
// pipeline sees every event published before the server stopped.
func (s *subscription[T]) Run(ctx context.Context) {
	for {
		select {
		case event := <-s.events:
			s.deliver(ctx, event)
		case <-ctx.Done():
			s.drain(context.WithoutCancel(ctx))
			return
		}
	}
}

func (s *subscription[T]) drain(ctx context.Context) {
	for {
		select {
		case event := <-s.events:
			s.deliver(ctx, event)
		default:
			return
		}
	}
}

func (s *subscription[T]) deliver(ctx context.Context, event proto.Message) {
	typed, ok := event.(T)
	if !ok {
		return
	}
	_ = s.handler.Handle(ctx, typed)
}

// failures wraps a handler so the bus counts what it refused.
type failures[T proto.Message] struct {
	inner  Handler[T]
	failed prometheus.Counter
}

func (f failures[T]) Handle(ctx context.Context, event T) error {
	err := f.inner.Handle(ctx, event)
	if err != nil {
		f.failed.Inc()
	}
	return err //nolint:wrapcheck // a decorator adds a count, not a sentence.
}

var errSealed = errors.New("subscribed after the runners started: subscribe while the module is built")

type subscriber struct {
	name    string
	events  chan proto.Message
	dropped prometheus.Counter
}

type eventBus struct {
	mu          sync.RWMutex
	sealed      bool
	subscribers map[protoreflect.FullName][]subscriber

	dropped *prometheus.CounterVec
	failed  *prometheus.CounterVec
}

func newEventBus(metrics prometheus.Registerer) *eventBus {
	return &eventBus{
		subscribers: map[protoreflect.FullName][]subscriber{},
		dropped: promauto.With(metrics).NewCounterVec(prometheus.CounterOpts{
			Name: "events_dropped_total",
			Help: "Events a subscriber missed because its buffer was full",
		}, []string{"event", "subscriber"}),
		failed: promauto.With(metrics).NewCounterVec(prometheus.CounterOpts{
			Name: "events_failed_total",
			Help: "Events a subscriber's handler answered with an error",
		}, []string{"event", "subscriber"}),
	}
}

// Publish hands every subscriber its own copy: the first gets the message, the others a clone.
func (b *eventBus) Publish(event proto.Message) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	for i, subscriber := range b.subscribers[event.ProtoReflect().Descriptor().FullName()] {
		copied := event
		if i > 0 {
			copied = proto.Clone(event)
		}

		select {
		case subscriber.events <- copied:
		default:
			subscriber.dropped.Inc()
		}
	}
}

func (b *eventBus) register(
	event protoreflect.FullName,
	name string,
	buffer int,
) (<-chan proto.Message, prometheus.Counter, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.sealed {
		return nil, nil, fmt.Errorf("%s on %s: %w", name, event, errSealed)
	}
	if buffer <= 0 {
		return nil, nil, fmt.Errorf("%s on %s: the buffer must hold at least one event", name, event)
	}
	for _, existing := range b.subscribers[event] {
		if existing.name == name {
			return nil, nil, fmt.Errorf("%s already subscribes to %s", name, event)
		}
	}

	events := make(chan proto.Message, buffer)
	b.subscribers[event] = append(b.subscribers[event], subscriber{
		name:    name,
		events:  events,
		dropped: b.dropped.WithLabelValues(string(event), name),
	})

	return events, b.failed.WithLabelValues(string(event), name), nil
}

// seal refuses every later subscriber. It runs once every module is built, before the runners start.
func (b *eventBus) seal() {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.sealed = true
}
