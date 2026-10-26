package cpredis

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/redis/go-redis/v9"
	"github.com/redis/go-redis/v9/maintnotifications"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpbootstrap"
)

type Config struct {
	Address  string
	Password string

	MaxLen         int64
	PublishTimeout time.Duration
	RetryAfter     time.Duration
	MaxDeliveries  int
}

// String leaves out the password: the config is logged at boot.
func (c Config) String() string {
	return fmt.Sprintf("{Address:%s MaxLen:%d PublishTimeout:%s RetryAfter:%s MaxDeliveries:%d}",
		c.Address, c.MaxLen, c.PublishTimeout, c.RetryAfter, c.MaxDeliveries)
}

func (c Config) Validate() error {
	var errs []error
	if _, port, err := net.SplitHostPort(c.Address); err != nil || port == "" {
		errs = append(errs, fmt.Errorf("address %q is not a host:port", c.Address))
	}
	if c.MaxLen < 0 || c.MaxDeliveries < 0 || c.PublishTimeout < 0 || c.RetryAfter < 0 {
		errs = append(errs, errors.New("maxLen, maxDeliveries, publishTimeout and retryAfter cannot be negative"))
	}
	return errors.Join(errs...)
}

const (
	defaultMaxLen         = 100_000
	defaultPublishTimeout = 500 * time.Millisecond
	defaultRetryAfter     = 2 * time.Second
	defaultMaxDeliveries  = 8
)

func (c Config) withDefaults() Config {
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
	fieldPayload = "payload"
	poolSize     = 32
)

func streamOf(event protoreflect.FullName) string { return "events:" + string(event) }

func New(config Config, metrics prometheus.Registerer) *Streams {
	return &Streams{config: config.withDefaults(), metrics: metrics, groups: map[protoreflect.FullName][]string{}}
}

type Streams struct {
	config  Config
	metrics prometheus.Registerer

	client      *redis.Client
	unpublished *prometheus.CounterVec

	mu     sync.Mutex
	groups map[protoreflect.FullName][]string
}

func (s *Streams) ConnectCtx(ctx context.Context) error {
	if err := s.config.Validate(); err != nil {
		return err
	}

	unpublished, err := shared(s.metrics, prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "events_unpublished_total",
		Help: "Events lost because redis did not take them",
	}, []string{"event"}))
	if err != nil {
		return err
	}

	client := redis.NewClient(&redis.Options{
		Addr:                     s.config.Address,
		Password:                 s.config.Password,
		PoolSize:                 poolSize,
		MaintNotificationsConfig: &maintnotifications.Config{Mode: maintnotifications.ModeDisabled},
	})
	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		return fmt.Errorf("ping redis at %s: %w", s.config.Address, err)
	}

	s.client = client
	s.unpublished = unpublished
	return nil
}

// Every module that publishes counts its losses under one name.
func shared(metrics prometheus.Registerer, vec *prometheus.CounterVec) (*prometheus.CounterVec, error) {
	err := metrics.Register(vec)
	var already prometheus.AlreadyRegisteredError
	if errors.As(err, &already) {
		if existing, ok := already.ExistingCollector.(*prometheus.CounterVec); ok {
			return existing, nil
		}
	}
	if err != nil {
		return nil, fmt.Errorf("failed to register events_unpublished_total: %w", err)
	}
	return vec, nil
}

func (s *Streams) Close() error {
	if s.client == nil {
		return nil
	}
	return s.client.Close() //nolint:wrapcheck // closing names nothing more.
}

func (s *Streams) Publish(events ...proto.Message) {
	ctx, cancel := context.WithTimeout(context.Background(), s.config.PublishTimeout)
	defer cancel()

	lost, _ := s.add(ctx, events)
	for _, event := range lost {
		s.unpublished.WithLabelValues(string(event.ProtoReflect().Descriptor().FullName())).Inc()
	}
}

func (s *Streams) Send(ctx context.Context, events ...proto.Message) error {
	if lost, err := s.add(ctx, events); len(lost) > 0 {
		return fmt.Errorf("redis took %d of %d events: %w", len(events)-len(lost), len(events), err)
	}
	return nil
}

// One round trip; each stream gets its events in the order given.
func (s *Streams) add(ctx context.Context, events []proto.Message) ([]proto.Message, error) {
	type added struct {
		event proto.Message
		cmd   *redis.StringCmd
	}

	var (
		lost  []proto.Message
		adds  []added
		first error
	)
	pipe := s.client.Pipeline()
	for _, event := range events {
		payload, err := proto.Marshal(event)
		if err != nil {
			lost, first = append(lost, event), cmp.Or(first, err)
			continue
		}
		adds = append(adds, added{event: event, cmd: pipe.XAdd(ctx, &redis.XAddArgs{
			Stream: streamOf(event.ProtoReflect().Descriptor().FullName()),
			MaxLen: s.config.MaxLen,
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

func (s *Streams) claim(event protoreflect.FullName, name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, existing := range s.groups[event] {
		if existing == name {
			return fmt.Errorf("%s already subscribes to %s", name, event)
		}
	}
	s.groups[event] = append(s.groups[event], name)
	return nil
}

func CloseAfter(streams *Streams, logger *slog.Logger, runners ...cpbootstrap.Runner) cpbootstrap.Runner {
	return closeAfter{runners: runners, streams: streams, logger: logger}
}

type closeAfter struct {
	runners []cpbootstrap.Runner
	streams *Streams
	logger  *slog.Logger
}

func (c closeAfter) Name() string {
	names := make([]string, len(c.runners))
	for i, runner := range c.runners {
		names[i] = runner.Name()
	}
	return strings.Join(names, "+")
}

func (c closeAfter) Run(ctx context.Context) {
	var wg sync.WaitGroup
	for _, runner := range c.runners {
		wg.Go(func() { runner.Run(ctx) })
	}
	wg.Wait()

	if err := c.streams.Close(); err != nil {
		c.logger.Error("failed to close a redis client", slog.String("runner", c.Name()), slog.Any("error", err))
	}
}
