package cpredis_test

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"runtime/debug"
	"sync"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpbootstrap"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpredis"
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

func connected(t *testing.T, config cpredis.Config, metrics prometheus.Registerer) *cpredis.Streams {
	t.Helper()

	streams := cpredis.New(config, metrics)
	require.NoError(t, streams.ConnectCtx(t.Context()))
	t.Cleanup(func() { _ = streams.Close() })
	return streams
}

func subscribed[T proto.Message](t *testing.T, streams *cpredis.Streams, name string, handler cpredis.Handler[T]) cpbootstrap.Runner {
	t.Helper()

	runner, err := cpredis.Subscribe(t.Context(), streams, name, handler)
	require.NoError(t, err)
	return runner
}

func running(runners ...cpbootstrap.Runner) func() {
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	for _, runner := range runners {
		wg.Go(func() { runner.Run(ctx) })
	}
	return func() {
		cancel()
		wg.Wait()
	}
}

func publish(streams *cpredis.Streams, values ...string) {
	events := make([]proto.Message, len(values))
	for i, value := range values {
		events[i] = wrapperspb.String(value)
	}
	streams.Publish(events...)
}

func numbered(n int) []string {
	values := make([]string, n)
	for i := range values {
		values[i] = fmt.Sprint(i)
	}
	return values
}

func eventually(t *testing.T, condition func() bool) {
	t.Helper()

	require.Eventually(t, condition, 5*time.Second, 10*time.Millisecond)
}

func gauge(t *testing.T, metrics *prometheus.Registry, name, subscriber string) (float64, bool) {
	t.Helper()

	families, err := metrics.Gather()
	require.NoError(t, err)
	for _, family := range families {
		if family.GetName() != name {
			continue
		}
		for _, metric := range family.GetMetric() {
			for _, label := range metric.GetLabel() {
				if label.GetName() == "subscriber" && label.GetValue() == subscriber {
					return metric.GetCounter().GetValue() + metric.GetGauge().GetValue(), true
				}
			}
		}
	}
	return 0, false
}

func TestEveryModuleReadsEveryEventInOrderThoughItConnectedAfterThePublish(t *testing.T) {
	server := cpredis.StartTestServer(t)
	publish(connected(t, server.Config(), prometheus.NewRegistry()), numbered(300)...)
	first, second := &received{}, &received{}

	stop := running(
		subscribed[*wrapperspb.StringValue](t, connected(t, server.Config(), prometheus.NewRegistry()), "player-first", first),
		subscribed[*wrapperspb.StringValue](t, connected(t, server.Config(), prometheus.NewRegistry()), "chat-second", second),
	)
	eventually(t, func() bool { return len(first.values()) == 300 && len(second.values()) == 300 })
	stop()

	assert.Equal(t, numbered(300), first.values(), "in the order they were published")
	assert.Equal(t, numbered(300), second.values())
}

func TestEachSubscriberGetsItsOwnCopy(t *testing.T) {
	streams := connected(t, cpredis.StartTestServer(t).Config(), prometheus.NewRegistry())
	first, second := &received{}, &received{}
	stop := running(subscribed[*wrapperspb.StringValue](t, streams, "first", first),
		subscribed[*wrapperspb.StringValue](t, streams, "second", second))

	publish(streams, "taken")
	eventually(t, func() bool { return len(first.values()) == 1 && len(second.values()) == 1 })
	stop()

	firstEvents, _ := first.snapshot()
	secondEvents, _ := second.snapshot()
	assert.NotSame(t, firstEvents[0], secondEvents[0])
	assert.Equal(t, "taken", secondEvents[0].GetValue())
}

func TestNoSubscriberCodeRunsInThePublishersStack(t *testing.T) {
	streams := connected(t, cpredis.StartTestServer(t).Config(), prometheus.NewRegistry())
	subscriber := &received{}
	stop := running(subscribed[*wrapperspb.StringValue](t, streams, "subscriber", subscriber))

	publish(streams, "taken")
	eventually(t, func() bool { return len(subscriber.values()) == 1 })
	stop()

	_, stacks := subscriber.snapshot()
	assert.NotContains(t, stacks[0], "Publish")
	assert.Contains(t, stacks[0], "Run", "delivered by the subscriber's runner")
}

func TestAnEventOfAnotherTypeIsNotDelivered(t *testing.T) {
	streams := connected(t, cpredis.StartTestServer(t).Config(), prometheus.NewRegistry())
	subscriber := &received{}
	numbers := make(chan int64, 1)
	stop := running(subscribed[*wrapperspb.StringValue](t, streams, "strings", subscriber),
		subscribed(t, streams, "numbers", handlerFunc[*wrapperspb.Int64Value](func(_ context.Context, event *wrapperspb.Int64Value) error {
			numbers <- event.GetValue()
			return nil
		})))

	streams.Publish(wrapperspb.Int64(7))
	assert.Equal(t, int64(7), <-numbers)
	stop()

	assert.Empty(t, subscriber.values())
}

func TestTwoSubscribersOfOneTypeCannotShareAName(t *testing.T) {
	streams := connected(t, cpredis.StartTestServer(t).Config(), prometheus.NewRegistry())
	subscribed[*wrapperspb.StringValue](t, streams, "player", &received{})

	_, err := cpredis.Subscribe[*wrapperspb.StringValue](t.Context(), streams, "player", &received{})

	assert.ErrorContains(t, err, "already subscribes")
}

func TestAModuleConnectsOnlyToARedisItNamesAndReaches(t *testing.T) {
	assert.ErrorContains(t, cpredis.New(cpredis.Config{}, prometheus.NewRegistry()).ConnectCtx(t.Context()), "host:port")

	ctx, cancel := context.WithTimeout(t.Context(), time.Second)
	defer cancel()
	assert.ErrorContains(t, cpredis.New(cpredis.Config{Address: "127.0.0.1:1"}, prometheus.NewRegistry()).ConnectCtx(ctx), "ping")
}

func TestEveryModuleCountsWhatRedisRefusedUnderOneName(t *testing.T) {
	server := cpredis.StartTestServer(t)
	metrics := prometheus.NewRegistry()
	planet, chat := connected(t, server.Config(), metrics), connected(t, server.Config(), metrics)
	require.NoError(t, planet.Close())
	require.NoError(t, chat.Close())

	publish(planet, "taken")
	publish(chat, "sent")

	families, err := metrics.Gather()
	require.NoError(t, err)
	require.Len(t, families, 1)
	assert.Equal(t, "events_unpublished_total", families[0].GetName())
	assert.InDelta(t, 2, families[0].GetMetric()[0].GetCounter().GetValue(), 0)
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
	metrics := prometheus.NewRegistry()
	streams := connected(t, cpredis.StartTestServer(t).Config(), metrics)
	handler := &flaky{fails: map[string]int{"first": 2}}
	stop := running(subscribed[*wrapperspb.StringValue](t, streams, "flaky", handler))

	publish(streams, "first", "second")
	eventually(t, func() bool { return len(handler.tries()) == 4 })
	eventually(t, func() bool {
		pending, _ := gauge(t, metrics, "events_pending", "flaky")
		lag, _ := gauge(t, metrics, "events_lag", "flaky")
		return pending == 0 && lag == 0
	})
	stop()

	assert.Equal(t, []string{"first", "first", "first", "second"}, handler.tries())
	failed, _ := gauge(t, metrics, "events_failed_total", "flaky")
	assert.InDelta(t, 2, failed, 0)
}

func TestAnEventThatKeepsFailingIsBuriedWhereAnOperatorCanReadIt(t *testing.T) {
	metrics := prometheus.NewRegistry()
	config := cpredis.StartTestServer(t).Config()
	config.MaxDeliveries = 3
	streams := connected(t, config, metrics)
	handler := &flaky{fails: map[string]int{"poison": 100}}
	stop := running(subscribed[*wrapperspb.StringValue](t, streams, "failing", handler))

	publish(streams, "poison", "next")
	eventually(t, func() bool { return len(handler.tries()) == 4 })
	stop()

	assert.Equal(t, []string{"poison", "poison", "poison", "next"}, handler.tries(), "maxDeliveries tries, then the next one")
	dead, _ := gauge(t, metrics, "events_dead_total", "failing")
	assert.InDelta(t, 1, dead, 0)

	client := redis.NewClient(&redis.Options{Addr: config.Address})
	t.Cleanup(func() { _ = client.Close() })
	buried, err := client.XRange(t.Context(), "events:dead", "-", "+").Result()
	require.NoError(t, err)
	require.Len(t, buried, 1)
	assert.Equal(t, "google.protobuf.StringValue", buried[0].Values["event"])
	assert.Equal(t, "failing", buried[0].Values["subscriber"])
	assert.Equal(t, "3", buried[0].Values["deliveries"])
}

func TestWhatACrashLeftPendingIsHandledFirstByTheNextProcess(t *testing.T) {
	config := cpredis.StartTestServer(t).Config()
	crashed := make(chan struct{})
	before := connected(t, config, prometheus.NewRegistry())
	stop := running(subscribed(t, before, "player",
		handlerFunc[*wrapperspb.StringValue](func(context.Context, *wrapperspb.StringValue) error {
			close(crashed)
			runtime.Goexit()
			return nil
		})))
	publish(before, "first", "second")
	<-crashed
	stop()

	after := connected(t, config, prometheus.NewRegistry())
	again := &received{}
	stop = running(subscribed[*wrapperspb.StringValue](t, after, "player", again))
	publish(after, "third")
	eventually(t, func() bool { return len(again.values()) == 3 })
	stop()

	assert.Equal(t, []string{"first", "second", "third"}, again.values(),
		"a handler that ran before the crash runs again: it must be safe to run twice")
}

func TestWhatWasReadBeforeShutdownIsStillHandled(t *testing.T) {
	streams := connected(t, cpredis.StartTestServer(t).Config(), prometheus.NewRegistry())
	subscriber := &received{}
	release := make(chan struct{})
	reading := make(chan struct{})
	var once sync.Once
	stop := running(subscribed(t, streams, "blocked",
		handlerFunc[*wrapperspb.StringValue](func(ctx context.Context, event *wrapperspb.StringValue) error {
			once.Do(func() { close(reading) })
			<-release
			return subscriber.Handle(ctx, event)
		})))

	publish(streams, numbered(10)...)
	<-reading
	go func() {
		time.Sleep(50 * time.Millisecond)
		close(release)
	}()
	stop()

	assert.Equal(t, numbered(10), subscriber.values())
}
