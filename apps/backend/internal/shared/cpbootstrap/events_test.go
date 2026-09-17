package cpbootstrap_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"runtime/debug"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpbootstrap"
)

type handlerFunc[T any] func(ctx context.Context, event T) error

func (f handlerFunc[T]) Handle(ctx context.Context, event T) error { return f(ctx, event) }

// received keeps what one subscriber was handed, in order.
type received struct {
	mu     sync.Mutex
	events []*wrapperspb.StringValue
	stacks []string
}

func (r *received) Handle(_ context.Context, event *wrapperspb.StringValue) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.events = append(r.events, event)
	r.stacks = append(r.stacks, string(debug.Stack()))
	return nil
}

func (r *received) values() []string {
	r.mu.Lock()
	defer r.mu.Unlock()

	values := make([]string, len(r.events))
	for i, event := range r.events {
		values[i] = event.GetValue()
	}
	return values
}

func (r *received) snapshot() ([]*wrapperspb.StringValue, []string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	return append([]*wrapperspb.StringValue(nil), r.events...), append([]string(nil), r.stacks...)
}

func subscriberModule(name string, buffer int, handler cpbootstrap.Handler[*wrapperspb.StringValue]) cpbootstrap.Module {
	return newModule(name, func(props cpbootstrap.Props) error {
		runner, err := cpbootstrap.Subscribe(props.Events, name, buffer, handler)
		if err != nil {
			return err //nolint:wrapcheck // the test reads the message.
		}
		props.Runners.Add(runner)
		return nil
	})
}

func publisherModule(values ...string) cpbootstrap.Module {
	return newModule("publisher", func(props cpbootstrap.Props) error {
		for _, value := range values {
			props.Events.Publish(wrapperspb.String(value))
		}
		return nil
	})
}

func numbered(n int) []string {
	values := make([]string, n)
	for i := range values {
		values[i] = fmt.Sprint(i)
	}
	return values
}

func servedMetrics(t *testing.T, address string) string {
	t.Helper()

	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "http://"+address+"/metrics", nil)
	require.NoError(t, err)
	res, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer func() { _ = res.Body.Close() }()
	body, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	return string(body)
}

func TestEventsPublishedWhileModulesAreBuiltReachASubscriberRegisteredBefore(t *testing.T) {
	first, second := &received{}, &received{}
	server := cpbootstrap.ServerConfig{BindAddress: freeAddress(t)}

	serveUntil(t, server, func() {
		require.Eventually(t, func() bool { return len(first.values()) == 100 && len(second.values()) == 100 },
			time.Second, time.Millisecond)
	}, subscriberModule("first", 100, first), subscriberModule("second", 100, second), publisherModule(numbered(100)...))

	assert.Equal(t, numbered(100), first.values(), "in the order they were published")
	assert.Equal(t, numbered(100), second.values())
}

func TestEachSubscriberGetsItsOwnCopy(t *testing.T) {
	first, second := &received{}, &received{}
	server := cpbootstrap.ServerConfig{BindAddress: freeAddress(t)}

	serveUntil(t, server, func() {
		require.Eventually(t, func() bool { return len(first.values()) == 1 && len(second.values()) == 1 },
			time.Second, time.Millisecond)
	}, subscriberModule("first", 1, first), subscriberModule("second", 1, second), publisherModule("taken"))

	firstEvents, _ := first.snapshot()
	secondEvents, _ := second.snapshot()
	assert.NotSame(t, firstEvents[0], secondEvents[0])
	assert.Equal(t, "taken", secondEvents[0].GetValue())
}

func TestAFullBufferDropsTheEventAndCountsIt(t *testing.T) {
	slow := &received{}
	server := cpbootstrap.ServerConfig{BindAddress: freeAddress(t)}

	serveUntil(t, server, func() {
		require.Eventually(t, func() bool { return len(slow.values()) == 2 }, time.Second, time.Millisecond)
		assert.Contains(t, servedMetrics(t, server.BindAddress),
			`events_dropped_total{event="google.protobuf.StringValue",subscriber="slow"} 3`)
	}, subscriberModule("slow", 2, slow), publisherModule(numbered(5)...))

	assert.Equal(t, []string{"0", "1"}, slow.values(), "the first two fit, the rest were dropped rather than waited on")
}

func TestNoSubscriberCodeRunsInThePublishersStack(t *testing.T) {
	subscriber := &received{}
	server := cpbootstrap.ServerConfig{BindAddress: freeAddress(t)}
	published := make(chan struct{})

	publisher := newModule("publisher", func(props cpbootstrap.Props) error {
		props.Runners.Add(runner{name: "publisher", run: func(context.Context) {
			props.Events.Publish(wrapperspb.String("taken"))
			close(published)
		}})
		return nil
	})

	serveUntil(t, server, func() {
		<-published
		require.Eventually(t, func() bool { return len(subscriber.values()) == 1 }, time.Second, time.Millisecond)
	}, subscriberModule("subscriber", 1, subscriber), publisher)

	_, stacks := subscriber.snapshot()
	assert.NotContains(t, stacks[0], "Publish")
	assert.Contains(t, stacks[0], "Run", "delivered by the subscriber's runner")
}

func TestSubscribingAfterTheRunnersStartIsRefused(t *testing.T) {
	server := cpbootstrap.ServerConfig{BindAddress: freeAddress(t)}
	refused := make(chan error, 1)

	late := newModule("late", func(props cpbootstrap.Props) error {
		props.Runners.Add(runner{name: "late", run: func(context.Context) {
			_, err := cpbootstrap.Subscribe(props.Events, "late", 1, &received{})
			refused <- err
		}})
		return nil
	})

	serveUntil(t, server, func() {
		assert.ErrorContains(t, <-refused, "subscribe while the module is built")
	}, late)
}

func TestAnEventOfAnotherTypeIsNotDelivered(t *testing.T) {
	subscriber := &received{}
	numbers := make(chan int64, 1)
	server := cpbootstrap.ServerConfig{BindAddress: freeAddress(t)}

	numberModule := newModule("numbers", func(props cpbootstrap.Props) error {
		runner, err := cpbootstrap.Subscribe(props.Events, "numbers", 1,
			handlerFunc[*wrapperspb.Int64Value](func(_ context.Context, event *wrapperspb.Int64Value) error {
				numbers <- event.GetValue()
				return nil
			}))
		if err != nil {
			return err //nolint:wrapcheck // the test reads the message.
		}
		props.Runners.Add(runner)
		props.Events.Publish(wrapperspb.Int64(7))
		return nil
	})

	serveUntil(t, server, func() {
		assert.Equal(t, int64(7), <-numbers)
	}, subscriberModule("strings", 1, subscriber), numberModule)

	assert.Empty(t, subscriber.values())
}

func TestTwoSubscribersOfOneTypeCannotShareAName(t *testing.T) {
	err := run(t, []cpbootstrap.Module{
		subscriberModule("player", 1, &received{}),
		subscriberModule("player", 1, &received{}),
	})

	assert.ErrorContains(t, err, "already subscribes")
}

func TestAHandlerErrorIsCounted(t *testing.T) {
	server := cpbootstrap.ServerConfig{BindAddress: freeAddress(t)}
	handled := make(chan struct{}, 1)
	failing := handlerFunc[*wrapperspb.StringValue](func(context.Context, *wrapperspb.StringValue) error {
		handled <- struct{}{}
		return errors.New("no such account")
	})

	serveUntil(t, server, func() {
		<-handled
		require.Eventually(t, func() bool {
			return strings.Contains(servedMetrics(t, server.BindAddress),
				`events_failed_total{event="google.protobuf.StringValue",subscriber="failing"} 1`)
		}, time.Second, 10*time.Millisecond)
	}, subscriberModule("failing", 1, failing), publisherModule("taken"))
}

func TestWhatIsBufferedAtShutdownIsStillDelivered(t *testing.T) {
	subscriber := &received{}
	server := cpbootstrap.ServerConfig{BindAddress: freeAddress(t)}
	release := make(chan struct{})

	blocked := handlerFunc[*wrapperspb.StringValue](func(ctx context.Context, event *wrapperspb.StringValue) error {
		<-release
		return subscriber.Handle(ctx, event)
	})

	serveUntil(t, server, func() {
		go func() {
			time.Sleep(50 * time.Millisecond)
			close(release)
		}()
	}, subscriberModule("blocked", 10, blocked), publisherModule(numbered(10)...))

	assert.Equal(t, numbered(10), subscriber.values())
}
