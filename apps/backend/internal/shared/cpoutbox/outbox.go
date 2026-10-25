package cpoutbox

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/lib/pq"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpbootstrap"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func New(db Conn) Outbox {
	return Outbox{db: db}
}

type Conn interface {
	cppg.Conn
	InOpenTx(ctx context.Context, do func(ctx context.Context) error) error
}

type Outbox struct {
	db Conn
}

// Record keeps events in the transaction ctx carries, and refuses outside one: an event is sent only with what it tells of.
func (o Outbox) Record(ctx context.Context, events ...proto.Message) error {
	return o.db.InOpenTx(ctx, func(ctx context.Context) error { //nolint:wrapcheck // the error names the event.
		return keep(ctx, o.db.On(ctx), events...)
	})
}

func keep(ctx context.Context, tx cppg.Querier, events ...proto.Message) error {
	for _, event := range events {
		name := event.ProtoReflect().Descriptor().FullName()
		payload, err := proto.Marshal(event)
		if err != nil {
			return fmt.Errorf("failed to encode %s: %w", name, err)
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO outbox (event, payload) VALUES ($1::outbox_event, $2)`,
			string(name), payload); err != nil {
			return fmt.Errorf("failed to keep %s in the outbox: %w", name, err)
		}
	}
	return nil
}

type Sender interface {
	Send(ctx context.Context, events ...proto.Message) error
}

const (
	batch    = 100
	interval = time.Second
)

func NewRelay(name string, db cppg.Conn, events Sender, logger *slog.Logger) cpbootstrap.Runner {
	return runner{name: name, relay: logged{inner: relay{db: db, events: events}, logger: logger.With(slog.String("outbox", name))}}
}

type executor interface {
	Execute(ctx context.Context) (int, error)
}

type relay struct {
	db     cppg.Conn
	events Sender
}

// The rows read stay locked until they are sent and deleted, or the transaction rolls back.
func (r relay) Execute(ctx context.Context) (int, error) {
	var (
		sent   int
		unread []error
	)
	err := r.db.InTx(ctx, func(ctx context.Context) error {
		tx := r.db.On(ctx)
		rows, err := tx.QueryContext(ctx,
			`SELECT position, event::text, payload FROM outbox ORDER BY position LIMIT $1 FOR UPDATE SKIP LOCKED`, batch)
		if err != nil {
			return fmt.Errorf("failed to read the outbox: %w", err)
		}
		positions, events, dropped, err := read(rows)
		if err != nil {
			return err
		}
		if len(positions) == 0 {
			return nil
		}

		if err := r.events.Send(ctx, events...); err != nil {
			return fmt.Errorf("failed to send %d events of the outbox: %w", len(events), err)
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM outbox WHERE position = ANY($1)`, pq.Array(positions)); err != nil {
			return fmt.Errorf("failed to clear what the outbox sent: %w", err)
		}
		sent, unread = len(positions), dropped
		return nil
	})
	if err != nil {
		return 0, err
	}
	return sent, errors.Join(unread...)
}

// A row that does not read back is cleared with the others and reported: kept, it would hold every row behind it.
func read(rows *sql.Rows) ([]int64, []proto.Message, []error, error) {
	defer func() { _ = rows.Close() }()

	var (
		positions []int64
		events    []proto.Message
		unread    []error
	)
	for rows.Next() {
		var (
			position int64
			name     string
			payload  []byte
		)
		if err := rows.Scan(&position, &name, &payload); err != nil {
			return nil, nil, nil, fmt.Errorf("failed to read a row of the outbox: %w", err)
		}
		positions = append(positions, position)

		event, err := eventOf(protoreflect.FullName(name), payload)
		if err != nil {
			unread = append(unread, fmt.Errorf("row %d dropped: %w", position, err))
			continue
		}
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, nil, fmt.Errorf("failed to read the outbox: %w", err)
	}

	return positions, events, unread, nil
}

func eventOf(name protoreflect.FullName, payload []byte) (proto.Message, error) {
	kind, err := protoregistry.GlobalTypes.FindMessageByName(name)
	if err != nil {
		return nil, fmt.Errorf("no message is named %s: %w", name, err)
	}
	event := kind.New().Interface()
	if err := proto.Unmarshal(payload, event); err != nil {
		return nil, fmt.Errorf("failed to decode %s: %w", name, err)
	}
	return event, nil
}

type logged struct {
	inner  executor
	logger *slog.Logger
}

func (l logged) Execute(ctx context.Context) (int, error) {
	sent, err := l.inner.Execute(ctx)
	if err != nil {
		l.logger.Error("the outbox did not send everything", slog.Int("sent", sent), slog.Any("error", err))
	}
	return sent, err //nolint:wrapcheck // a decorator adds a log line, not a sentence.
}

type runner struct {
	name  string
	relay executor
}

func (r runner) Name() string { return r.name }

func (r runner) Run(ctx context.Context) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// The shutdown does not cut a batch off between its send and its delete, which would send it twice.
	working := context.WithoutCancel(ctx)
	for {
		r.empty(working)

		select {
		case <-ticker.C:
		case <-ctx.Done():
			r.empty(working)
			return
		}
	}
}

func (r runner) empty(ctx context.Context) {
	for {
		sent, err := r.relay.Execute(ctx)
		if err != nil || sent < batch {
			return
		}
	}
}
