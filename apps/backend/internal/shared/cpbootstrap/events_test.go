package cpbootstrap_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"runtime"
	"runtime/debug"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpbootstrap"
)

type handlerFunc[T any] func(ctx context.Context, event T) error

func (f handlerFunc[T]) Handle(ctx context.Context, event T) error { return f(ctx, event) }

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

func subscriberModule(name string, handler cpbootstrap.Handler[*wrapperspb.StringValue]) cpbootstrap.Module {
	return newModule(name, func(props cpbootstrap.Props) error {
		runner, err := cpbootstrap.Subscribe(props.Events, name, handler)
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

func serveEvents(t *testing.T, events cpbootstrap.EventsConfig, probe func(address string), modules ...cpbootstrap.Module) {
	t.Helper()

	public := listen(t)
	server := cpbootstrap.ServerConfig{BindAddress: public.Addr().String()}

	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() {
		done <- cpbootstrap.RunOn(ctx, cpbootstrap.Options{
			Server:  server,
			Events:  events,
			Logger:  slog.New(slog.DiscardHandler),
			Modules: modules,
		}, public)
	}()

	waitUntilServed(t, server.BindAddress)
	probe(server.BindAddress)

	cancel()
	require.NoError(t, <-done)
}

func eventually(t *testing.T, condition func() bool) {
	t.Helper()

	require.Eventually(t, condition, 5*time.Second, 10*time.Millisecond)
}

func TestEventsPublishedWhileModulesAreBuiltReachEverySubscriberInOrder(t *testing.T) {
	first, second := &received{}, &received{}

	serveEvents(t, cpbootstrap.StartTestRedis(t), func(string) {
		eventually(t, func() bool { return len(first.values()) == 300 && len(second.values()) == 300 })
	}, subscriberModule("first", first), publisherModule(numbered(300)...), subscriberModule("second", second))

	assert.Equal(t, numbered(300), first.values(), "in the order they were published")
	assert.Equal(t, numbered(300), second.values(), "a subscriber built after the publisher reads them too")
}

func TestEachSubscriberGetsItsOwnCopy(t *testing.T) {
	first, second := &received{}, &received{}

	serveEvents(t, cpbootstrap.StartTestRedis(t), func(string) {
		eventually(t, func() bool { return len(first.values()) == 1 && len(second.values()) == 1 })
	}, subscriberModule("first", first), subscriberModule("second", second), publisherModule("taken"))

	firstEvents, _ := first.snapshot()
	secondEvents, _ := second.snapshot()
	assert.NotSame(t, firstEvents[0], secondEvents[0])
	assert.Equal(t, "taken", secondEvents[0].GetValue())
}

func TestNoSubscriberCodeRunsInThePublishersStack(t *testing.T) {
	subscriber := &received{}
	published := make(chan struct{})

	publisher := newModule("publisher", func(props cpbootstrap.Props) error {
		props.Runners.Add(runner{name: "publisher", run: func(context.Context) {
			props.Events.Publish(wrapperspb.String("taken"))
			close(published)
		}})
		return nil
	})

	serveEvents(t, cpbootstrap.StartTestRedis(t), func(string) {
		<-published
		eventually(t, func() bool { return len(subscriber.values()) == 1 })
	}, subscriberModule("subscriber", subscriber), publisher)

	_, stacks := subscriber.snapshot()
	assert.NotContains(t, stacks[0], "Publish")
	assert.Contains(t, stacks[0], "Run", "delivered by the subscriber's runner")
}

func TestSubscribingAfterTheRunnersStartIsRefused(t *testing.T) {
	refused := make(chan error, 1)

	late := newModule("late", func(props cpbootstrap.Props) error {
		props.Runners.Add(runner{name: "late", run: func(context.Context) {
			_, err := cpbootstrap.Subscribe(props.Events, "late", &received{})
			refused <- err
		}})
		return nil
	})

	serveEvents(t, cpbootstrap.StartTestRedis(t), func(string) {
		assert.ErrorContains(t, <-refused, "subscribe while the module is built")
	}, late)
}

func TestSubscribingWithNoRedisRefusesTheBoot(t *testing.T) {
	err := run(t, []cpbootstrap.Module{subscriberModule("player", &received{})})

	assert.ErrorContains(t, err, "events.address is empty")
}

func TestAnEventOfAnotherTypeIsNotDelivered(t *testing.T) {
	subscriber := &received{}
	numbers := make(chan int64, 1)

	numberModule := newModule("numbers", func(props cpbootstrap.Props) error {
		runner, err := cpbootstrap.Subscribe(props.Events, "numbers",
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

	serveEvents(t, cpbootstrap.StartTestRedis(t), func(string) {
		assert.Equal(t, int64(7), <-numbers)
	}, subscriberModule("strings", subscriber), numberModule)

	assert.Empty(t, subscriber.values())
}

func TestTwoSubscribersOfOneTypeCannotShareAName(t *testing.T) {
	err := runWith(t, cpbootstrap.StartTestRedis(t), []cpbootstrap.Module{
		subscriberModule("player", &received{}),
		subscriberModule("player", &received{}),
	})

	assert.ErrorContains(t, err, "already subscribes")
}

type flaky struct {
	mu      sync.Mutex
	fails   map[string]int
	handled []string
}

func (f *flaky) Handle(_ context.Context, event *wrapperspb.StringValue) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.handled = append(f.handled, event.GetValue())
	if f.fails[event.GetValue()] > 0 {
		f.fails[event.GetValue()]--
		return errors.New("postgres is restarting")
	}
	return nil
}

func (f *flaky) tries() []string {
	f.mu.Lock()
	defer f.mu.Unlock()

	return append([]string(nil), f.handled...)
}

func TestAFailedEventIsTriedAgainBeforeTheOnesBehindIt(t *testing.T) {
	handler := &flaky{fails: map[string]int{"first": 2}}

	serveEvents(t, cpbootstrap.StartTestRedis(t), func(address string) {
		eventually(t, func() bool { return len(handler.tries()) == 4 })
		eventually(t, func() bool {
			metrics := servedMetrics(t, address)
			return strings.Contains(metrics, `events_failed_total{event="google.protobuf.StringValue",subscriber="flaky"} 2`) &&
				strings.Contains(metrics, `events_pending{event="google.protobuf.StringValue",subscriber="flaky"} 0`) &&
				strings.Contains(metrics, `events_lag{event="google.protobuf.StringValue",subscriber="flaky"} 0`)
		})
	}, subscriberModule("flaky", handler), publisherModule("first", "second"))

	assert.Equal(t, []string{"first", "first", "first", "second"}, handler.tries())
}

func TestAnEventThatKeepsFailingIsBuriedWhereAnOperatorCanReadIt(t *testing.T) {
	events := cpbootstrap.StartTestRedis(t)
	events.MaxDeliveries = 3
	handler := &flaky{fails: map[string]int{"poison": 100}}

	serveEvents(t, events, func(address string) {
		eventually(t, func() bool { return len(handler.tries()) == 4 })
		eventually(t, func() bool {
			return strings.Contains(servedMetrics(t, address),
				`events_dead_total{event="google.protobuf.StringValue",subscriber="failing"} 1`)
		})
	}, subscriberModule("failing", handler), publisherModule("poison", "next"))

	assert.Equal(t, []string{"poison", "poison", "poison", "next"}, handler.tries(), "maxDeliveries tries, then the next one")

	client := redis.NewClient(&redis.Options{Addr: events.Address})
	t.Cleanup(func() { _ = client.Close() })
	dead, err := client.XRange(t.Context(), "events:dead", "-", "+").Result()
	require.NoError(t, err)
	require.Len(t, dead, 1)
	assert.Equal(t, "google.protobuf.StringValue", dead[0].Values["event"])
	assert.Equal(t, "failing", dead[0].Values["subscriber"])
	assert.Equal(t, "3", dead[0].Values["deliveries"])
}

func TestWhatACrashLeftPendingIsHandledFirstAtTheNextBoot(t *testing.T) {
	events := cpbootstrap.StartTestRedis(t)
	crashed := make(chan struct{})
	crashing := handlerFunc[*wrapperspb.StringValue](func(context.Context, *wrapperspb.StringValue) error {
		close(crashed)
		runtime.Goexit()
		return nil
	})

	serveEvents(t, events, func(string) { <-crashed },
		subscriberModule("player", crashing), publisherModule("first", "second"))

	again := &received{}
	serveEvents(t, events, func(string) {
		eventually(t, func() bool { return len(again.values()) == 3 })
	}, subscriberModule("player", again), publisherModule("third"))

	assert.Equal(t, []string{"first", "second", "third"}, again.values(),
		"a handler that ran before the crash runs again: it must be safe to run twice")
}

func TestWhatWasReadBeforeShutdownIsStillHandled(t *testing.T) {
	subscriber := &received{}
	release := make(chan struct{})

	blocked := handlerFunc[*wrapperspb.StringValue](func(ctx context.Context, event *wrapperspb.StringValue) error {
		<-release
		return subscriber.Handle(ctx, event)
	})

	serveEvents(t, cpbootstrap.StartTestRedis(t), func(string) {
		go func() {
			time.Sleep(50 * time.Millisecond)
			close(release)
		}()
	}, subscriberModule("blocked", blocked), publisherModule(numbered(10)...))

	assert.Equal(t, numbered(10), subscriber.values())
}
