package cpbootstrap

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/redis/go-redis/v9"
	"github.com/redis/go-redis/v9/maintnotifications"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type EventBus interface {
	Publish(events ...proto.Message)
	Send(ctx context.Context, events ...proto.Message) error

	register(event protoreflect.FullName, subscriber string) (*group, error)
}

type Handler[T proto.Message] interface {
	Handle(ctx context.Context, event T) error
}

func Subscribe[T proto.Message](bus EventBus, subscriber string, handler Handler[T]) (Runner, error) {
	var zero T
	group, err := bus.register(zero.ProtoReflect().Descriptor().FullName(), subscriber)
	if err != nil {
		return nil, err
	}

	return &subscription[T]{group: group, handler: handler}, nil
}

type EventsConfig struct {
	Address  string
	Password string

	MaxLen         int64
	PublishTimeout time.Duration
	RetryAfter     time.Duration
	MaxDeliveries  int
}

// String leaves out the password: the config is logged at boot.
func (c EventsConfig) String() string {
	return fmt.Sprintf("{Address:%s MaxLen:%d PublishTimeout:%s RetryAfter:%s MaxDeliveries:%d}",
		c.Address, c.MaxLen, c.PublishTimeout, c.RetryAfter, c.MaxDeliveries)
}

func (c EventsConfig) Validate() error {
	var errs []error
	if c.Address != "" && !hostPort(c.Address) {
		errs = append(errs, fmt.Errorf("events.address %q is not a host:port", c.Address))
	}
	if c.MaxLen < 0 || c.MaxDeliveries < 0 || c.PublishTimeout < 0 || c.RetryAfter < 0 {
		errs = append(errs, errors.New("events.maxLen, maxDeliveries, publishTimeout and retryAfter cannot be negative"))
	}
	return errors.Join(errs...)
}

const (
	defaultMaxLen         = 100_000
	defaultPublishTimeout = 500 * time.Millisecond
	defaultRetryAfter     = 2 * time.Second
	defaultMaxDeliveries  = 8
)

func (c EventsConfig) withDefaults() EventsConfig {
	if c.MaxLen <= 0 {
		c.MaxLen = defaultMaxLen
	}
	if c.PublishTimeout <= 0 {
		c.PublishTimeout = defaultPublishTimeout
	}
	if c.RetryAfter <= 0 {
		c.RetryAfter = defaultRetryAfter
	}
	if c.MaxDeliveries <= 0 {
		c.MaxDeliveries = defaultMaxDeliveries
	}
	return c
}

const (
	consumer     = "backend"
	fieldPayload = "payload"

	deadLetters   = "events:dead"
	deadMaxLen    = 10_000
	readCount     = 100
	block         = 500 * time.Millisecond
	noBlock       = -1
	drainReads    = 10
	trimInterval  = time.Minute
	scrapeTimeout = 2 * time.Second
	poolSize      = 64
	// The wait before a retry doubles six times at most: 64 × retryAfter.
	mostDoublings = 6
)

func streamOf(event protoreflect.FullName) string { return "events:" + string(event) }

var (
	errSealed   = errors.New("subscribed after the runners started: subscribe while the module is built")
	errNoBroker = errors.New("events.address is empty: there is no redis to subscribe on")
)

type eventBus struct {
	client *redis.Client
	config EventsConfig

	mu     sync.RWMutex
	sealed bool
	groups map[protoreflect.FullName][]*group

	unpublished *prometheus.CounterVec
	failed      *prometheus.CounterVec
	dead        *prometheus.CounterVec
}

func newEventBus(ctx context.Context, config EventsConfig, metrics prometheus.Registerer) (*eventBus, error) {
	bus := &eventBus{
		config: config.withDefaults(),
		groups: map[protoreflect.FullName][]*group{},
		unpublished: promauto.With(metrics).NewCounterVec(prometheus.CounterOpts{
			Name: "events_unpublished_total",
			Help: "Events lost because redis did not take them",
		}, []string{"event"}),
		failed: promauto.With(metrics).NewCounterVec(prometheus.CounterOpts{
			Name: "events_failed_total",
			Help: "Events a subscriber's handler answered with an error",
		}, []string{"event", "subscriber"}),
		dead: promauto.With(metrics).NewCounterVec(prometheus.CounterOpts{
			Name: "events_dead_total",
			Help: "Events a subscriber gave up on, kept in the events:dead stream",
		}, []string{"event", "subscriber"}),
	}
	if config.Address == "" {
		return bus, nil
	}

	client := redis.NewClient(&redis.Options{
		Addr:                     config.Address,
		Password:                 config.Password,
		PoolSize:                 poolSize,
		MaintNotificationsConfig: &maintnotifications.Config{Mode: maintnotifications.ModeDisabled},
	})
	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("failed to reach redis at events.address %q: %w", config.Address, err)
	}
	bus.client = client
	metrics.MustRegister(lags{bus: bus})

	return bus, nil
}

func (b *eventBus) on() bool { return b.client != nil }

func (b *eventBus) Publish(events ...proto.Message) {
	ctx, cancel := context.WithTimeout(context.Background(), b.config.PublishTimeout)
	defer cancel()

	lost, _ := b.add(ctx, events)
	for _, event := range lost {
		b.unpublished.WithLabelValues(string(event.ProtoReflect().Descriptor().FullName())).Inc()
	}
}

func (b *eventBus) Send(ctx context.Context, events ...proto.Message) error {
	if lost, err := b.add(ctx, events); len(lost) > 0 {
		return fmt.Errorf("redis took %d of %d events: %w", len(events)-len(lost), len(events), err)
	}
	return nil
}

// One round trip; each stream gets its events in the order given.
func (b *eventBus) add(ctx context.Context, events []proto.Message) ([]proto.Message, error) {
	type added struct {
		event proto.Message
		cmd   *redis.StringCmd
	}

	if !b.on() {
		return nil, nil
	}

	var (
		lost  []proto.Message
		adds  []added
		first error
	)
	pipe := b.client.Pipeline()
	for _, event := range events {
		name := event.ProtoReflect().Descriptor().FullName()
		if !b.heard(name) {
			continue
		}
		payload, err := proto.Marshal(event)
		if err != nil {
			lost, first = append(lost, event), cmp.Or(first, err)
			continue
		}
		adds = append(adds, added{event: event, cmd: pipe.XAdd(ctx, &redis.XAddArgs{
			Stream: streamOf(name),
			MaxLen: b.config.MaxLen,
			Approx: true,
			Values: []any{fieldPayload, payload},
		})})
	}
	if len(adds) == 0 {
		return lost, first
	}

	_, _ = pipe.Exec(ctx)
	for _, add := range adds {
		if err := add.cmd.Err(); err != nil {
			lost, first = append(lost, add.event), cmp.Or(first, err)
		}
	}
	return lost, first
}

func (b *eventBus) heard(event protoreflect.FullName) bool {
	b.mu.RLock()
	defer b.mu.RUnlock()

	return len(b.groups[event]) > 0
}

func (b *eventBus) register(event protoreflect.FullName, name string) (*group, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if !b.on() {
		return nil, fmt.Errorf("%s on %s: %w", name, event, errNoBroker)
	}
	if b.sealed {
		return nil, fmt.Errorf("%s on %s: %w", name, event, errSealed)
	}
	for _, existing := range b.groups[event] {
		if existing.name == name {
			return nil, fmt.Errorf("%s already subscribes to %s", name, event)
		}
	}

	subscribed := &group{
		client:        b.client,
		stream:        streamOf(event),
		event:         event,
		name:          name,
		retryAfter:    b.config.RetryAfter,
		maxDeliveries: b.config.MaxDeliveries,
		failed:        b.failed.WithLabelValues(string(event), name),
		dead:          b.dead.WithLabelValues(string(event), name),
	}
	b.groups[event] = append(b.groups[event], subscribed)

	return subscribed, nil
}

func (b *eventBus) subscribers() map[protoreflect.FullName][]string {
	b.mu.RLock()
	defer b.mu.RUnlock()

	names := make(map[protoreflect.FullName][]string, len(b.groups))
	for event, groups := range b.groups {
		for _, group := range groups {
			names[event] = append(names[event], group.name)
		}
	}
	return names
}

// From 0, so what a module published while the others were built is read too.
func (b *eventBus) seal(ctx context.Context) error {
	b.mu.Lock()
	b.sealed = true
	b.mu.Unlock()

	for event, names := range b.subscribers() {
		stream := streamOf(event)
		for _, name := range names {
			err := b.client.XGroupCreateMkStream(ctx, stream, name, "0").Err()
			if err != nil && !redis.HasErrorPrefix(err, "BUSYGROUP") {
				return fmt.Errorf("failed to create the group %s on %s: %w", name, stream, err)
			}
		}

		groups, err := b.client.XInfoGroups(ctx, stream).Result()
		if err != nil {
			return fmt.Errorf("failed to list the groups on %s: %w", stream, err)
		}
		// A group nobody reads any more would hold the stream's trim back forever.
		for _, group := range groups {
			if slices.Contains(names, group.Name) {
				continue
			}
			if err := b.client.XGroupDestroy(ctx, stream, group.Name).Err(); err != nil {
				return fmt.Errorf("failed to destroy the group %s on %s: %w", group.Name, stream, err)
			}
		}
	}

	return nil
}

func (b *eventBus) close() error {
	if !b.on() {
		return nil
	}
	return b.client.Close()
}

func (b *eventBus) trim(ctx context.Context) {
	for event := range b.subscribers() {
		stream := streamOf(event)
		if floor, ok := b.floor(ctx, stream); ok {
			_ = b.client.XTrimMinIDApprox(ctx, stream, floor, 0).Err()
		}
	}
}

func (b *eventBus) floor(ctx context.Context, stream string) (string, bool) {
	groups, err := b.client.XInfoGroups(ctx, stream).Result()
	if err != nil || len(groups) == 0 {
		return "", false
	}

	floor := ""
	for _, group := range groups {
		kept := group.LastDeliveredID
		if group.Pending > 0 {
			pending, err := b.client.XPending(ctx, stream, group.Name).Result()
			if err != nil {
				return "", false
			}
			kept = pending.Lower
		}
		if floor == "" || before(kept, floor) {
			floor = kept
		}
	}
	return floor, true
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

type trimmer struct {
	bus *eventBus
}

func (t trimmer) Name() string { return "events-trim" }

func (t trimmer) Run(ctx context.Context) {
	ticker := time.NewTicker(trimInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			t.bus.trim(ctx)
		case <-ctx.Done():
			return
		}
	}
}

var (
	lagDesc = prometheus.NewDesc("events_lag",
		"Events in a subscriber's stream it has not read yet", []string{"event", "subscriber"}, nil)
	pendingDesc = prometheus.NewDesc("events_pending",
		"Events a subscriber read and has not handled yet", []string{"event", "subscriber"}, nil)
)

type lags struct {
	bus *eventBus
}

func (l lags) Describe(descs chan<- *prometheus.Desc) {
	descs <- lagDesc
	descs <- pendingDesc
}

func (l lags) Collect(metrics chan<- prometheus.Metric) {
	ctx, cancel := context.WithTimeout(context.Background(), scrapeTimeout)
	defer cancel()

	for event, names := range l.bus.subscribers() {
		groups, err := l.bus.client.XInfoGroups(ctx, streamOf(event)).Result()
		if err != nil {
			continue
		}
		for _, group := range groups {
			if !slices.Contains(names, group.Name) {
				continue
			}
			if group.Lag >= 0 {
				metrics <- prometheus.MustNewConstMetric(lagDesc, prometheus.GaugeValue, float64(group.Lag), string(event), group.Name)
			}
			metrics <- prometheus.MustNewConstMetric(pendingDesc, prometheus.GaugeValue, float64(group.Pending), string(event), group.Name)
		}
	}
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

type subscription[T proto.Message] struct {
	group   *group
	handler Handler[T]
}

func (s *subscription[T]) Name() string { return "events:" + s.group.name }

func (s *subscription[T]) Run(ctx context.Context) {
	if !s.recover(ctx) {
		return
	}

	for ctx.Err() == nil {
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
