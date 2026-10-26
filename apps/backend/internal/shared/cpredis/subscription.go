package cpredis

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/redis/go-redis/v9"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpbootstrap"
)

type Handler[T proto.Message] interface {
	Handle(ctx context.Context, event T) error
}

const (
	consumer      = "backend"
	deadLetters   = "events:dead"
	deadMaxLen    = 10_000
	readCount     = 100
	block         = 500 * time.Millisecond
	noBlock       = -1
	drainReads    = 10
	trimEvery     = time.Minute
	scrapeTimeout = 2 * time.Second
	// The wait before a retry doubles six times at most: 64 × retryAfter.
	mostDoublings = 6
)

// From 0, so what another module published before this group existed is read too.
func Subscribe[T proto.Message](ctx context.Context, streams *Streams, subscriber string, handler Handler[T]) (cpbootstrap.Runner, error) {
	return subscribe(ctx, streams, streamOf, subscriber, handler)
}

const commandHandler = "handler"

// A command has one group, so the module that handles it reads each command once, and one sent before it listened too.
func Handle[T proto.Message](ctx context.Context, streams *Streams, handler Handler[T]) (cpbootstrap.Runner, error) {
	return subscribe(ctx, streams, commandStreamOf, commandHandler, handler)
}

func subscribe[T proto.Message](
	ctx context.Context, streams *Streams, streamOf func(protoreflect.FullName) string, name string, handler Handler[T],
) (cpbootstrap.Runner, error) {
	var zero T
	message := zero.ProtoReflect().Descriptor().FullName()
	if err := streams.claim(message, name); err != nil {
		return nil, err
	}

	stream := streamOf(message)
	err := streams.client.XGroupCreateMkStream(ctx, stream, name, "0").Err()
	if err != nil && !redis.HasErrorPrefix(err, "BUSYGROUP") {
		return nil, fmt.Errorf("failed to create the group %s on %s: %w", name, stream, err)
	}

	subscribed, err := newGroup(streams, stream, message, name)
	if err != nil {
		return nil, err
	}
	return &subscription[T]{group: subscribed, handler: handler}, nil
}

type group struct {
	client        *redis.Client
	stream        string
	event         protoreflect.FullName
	name          string
	retryAfter    time.Duration
	maxDeliveries int

	failed prometheus.Counter
	dead   prometheus.Counter
	lag    *prometheus.Desc
	held   *prometheus.Desc
}

func newGroup(streams *Streams, stream string, event protoreflect.FullName, name string) (*group, error) {
	labels := prometheus.Labels{"event": string(event), "subscriber": name}
	subscribed := &group{
		client:        streams.client,
		stream:        stream,
		event:         event,
		name:          name,
		retryAfter:    streams.config.RetryAfter,
		maxDeliveries: streams.config.MaxDeliveries,
		failed: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "events_failed_total", Help: "Events a subscriber's handler answered with an error", ConstLabels: labels,
		}),
		dead: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "events_dead_total", Help: "Events a subscriber gave up on, kept in the events:dead stream", ConstLabels: labels,
		}),
		lag:  prometheus.NewDesc("events_lag", "Events in a subscriber's stream it has not read yet", nil, labels),
		held: prometheus.NewDesc("events_pending", "Events a subscriber read and has not handled yet", nil, labels),
	}

	for _, collector := range []prometheus.Collector{subscribed.failed, subscribed.dead, subscribed} {
		if err := streams.metrics.Register(collector); err != nil {
			return nil, fmt.Errorf("failed to register the metrics of %s: %w", name, err)
		}
	}
	return subscribed, nil
}

func (g *group) Describe(descs chan<- *prometheus.Desc) {
	descs <- g.lag
	descs <- g.held
}

func (g *group) Collect(metrics chan<- prometheus.Metric) {
	ctx, cancel := context.WithTimeout(context.Background(), scrapeTimeout)
	defer cancel()

	groups, err := g.client.XInfoGroups(ctx, g.stream).Result()
	if err != nil {
		return
	}
	for _, info := range groups {
		if info.Name != g.name {
			continue
		}
		if info.Lag >= 0 {
			metrics <- prometheus.MustNewConstMetric(g.lag, prometheus.GaugeValue, float64(info.Lag))
		}
		metrics <- prometheus.MustNewConstMetric(g.held, prometheus.GaugeValue, float64(info.Pending))
	}
}

func (g *group) read(ctx context.Context, wait time.Duration) ([]redis.XMessage, error) {
	streams, err := g.client.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group:    g.name,
		Consumer: consumer,
		Streams:  []string{g.stream, ">"},
		Count:    readCount,
		Block:    wait,
	}).Result()
	if errors.Is(err, redis.Nil) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read %s for %s: %w", g.stream, g.name, err)
	}

	var messages []redis.XMessage
	for _, stream := range streams {
		messages = append(messages, stream.Messages...)
	}
	return messages, nil
}

// Whatever the group left pending, under any consumer, comes back in stream order.
func (g *group) pending(ctx context.Context, from string) ([]redis.XMessage, string, error) {
	messages, next, err := g.client.XAutoClaim(ctx, &redis.XAutoClaimArgs{
		Stream:   g.stream,
		Group:    g.name,
		Consumer: consumer,
		Start:    from,
		Count:    readCount,
	}).Result()
	if err != nil {
		return nil, "", fmt.Errorf("failed to take back what %s left pending: %w", g.name, err)
	}
	return messages, next, nil
}

func (g *group) backoff(attempt int) time.Duration {
	return g.retryAfter << min(attempt-1, mostDoublings)
}

func (g *group) ack(ctx context.Context, position string) {
	_ = g.client.XAck(ctx, g.stream, g.name, position).Err()
}

// Left pending on a failure: the next boot takes it back.
func (g *group) bury(ctx context.Context, message redis.XMessage, attempts int) {
	_, err := g.client.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
		pipe.XAdd(ctx, &redis.XAddArgs{
			Stream: deadLetters,
			MaxLen: deadMaxLen,
			Approx: true,
			Values: []any{
				"event", string(g.event), "subscriber", g.name, "entry", message.ID, "deliveries", attempts,
				fieldPayload, message.Values[fieldPayload],
			},
		})
		pipe.XAck(ctx, g.stream, g.name, message.ID)
		return nil
	})
	if err == nil {
		g.dead.Inc()
	}
}

// To the oldest entry a group of the stream, of any module, has not handled.
func (g *group) trim(ctx context.Context) {
	groups, err := g.client.XInfoGroups(ctx, g.stream).Result()
	if err != nil || len(groups) == 0 {
		return
	}

	floor := ""
	for _, info := range groups {
		kept := info.LastDeliveredID
		if info.Pending > 0 {
			pending, err := g.client.XPending(ctx, g.stream, info.Name).Result()
			if err != nil {
				return
			}
			kept = pending.Lower
		}
		if floor == "" || before(kept, floor) {
			floor = kept
		}
	}
	_ = g.client.XTrimMinIDApprox(ctx, g.stream, floor, 0).Err()
}

func before(position, other string) bool {
	ms, seq := positionOf(position)
	otherMs, otherSeq := positionOf(other)
	return ms < otherMs || (ms == otherMs && seq < otherSeq)
}

func positionOf(position string) (uint64, uint64) {
	ms, seq, _ := strings.Cut(position, "-")
	parsedMs, _ := strconv.ParseUint(ms, 10, 64)
	parsedSeq, _ := strconv.ParseUint(seq, 10, 64)
	return parsedMs, parsedSeq
}

type subscription[T proto.Message] struct {
	group   *group
	handler Handler[T]
	trimmed time.Time
}

func (s *subscription[T]) Name() string { return "events:" + s.group.name }

func (s *subscription[T]) Run(ctx context.Context) {
	if !s.recover(ctx) {
		return
	}

	for ctx.Err() == nil {
		if time.Since(s.trimmed) >= trimEvery {
			s.group.trim(ctx)
			s.trimmed = time.Now()
		}

		messages, err := s.group.read(ctx, block)
		if err != nil {
			pause(ctx, block)
			continue
		}
		if !s.deliver(ctx, messages, true) {
			return
		}
	}

	s.drain(context.WithoutCancel(ctx))
}

func (s *subscription[T]) recover(ctx context.Context) bool {
	from := "0-0"
	for ctx.Err() == nil {
		messages, next, err := s.group.pending(ctx, from)
		if err != nil {
			pause(ctx, block)
			continue
		}
		if !s.deliver(ctx, messages, true) {
			return false
		}
		if next == "0-0" {
			return true
		}
		from = next
	}
	return false
}

func (s *subscription[T]) drain(ctx context.Context) {
	for range drainReads {
		messages, err := s.group.read(ctx, noBlock)
		if err != nil || len(messages) == 0 || !s.deliver(ctx, messages, false) {
			return
		}
	}
}

// False when one was left pending: the rest must wait for it, or the stream's order is lost.
func (s *subscription[T]) deliver(ctx context.Context, messages []redis.XMessage, retrying bool) bool {
	for _, message := range messages {
		if !s.handle(ctx, message, retrying) {
			return false
		}
	}
	return true
}

func (s *subscription[T]) handle(ctx context.Context, message redis.XMessage, retrying bool) bool {
	acting := context.WithoutCancel(ctx)

	var zero T
	event, ok := zero.ProtoReflect().New().Interface().(T)
	payload, _ := message.Values[fieldPayload].(string)
	if !ok || proto.Unmarshal([]byte(payload), event) != nil {
		s.group.bury(acting, message, 0)
		return true
	}

	for attempt := 1; ; attempt++ {
		err := s.handler.Handle(acting, event)
		if err == nil {
			s.group.ack(acting, message.ID)
			return true
		}
		s.group.failed.Inc()

		if attempt >= s.group.maxDeliveries {
			s.group.bury(acting, message, attempt)
			return true
		}
		if !retrying || !pause(ctx, s.group.backoff(attempt)) {
			return false
		}
	}
}

func pause(ctx context.Context, wait time.Duration) bool {
	timer := time.NewTimer(wait)
	defer timer.Stop()

	select {
	case <-timer.C:
		return true
	case <-ctx.Done():
		return false
	}
}
