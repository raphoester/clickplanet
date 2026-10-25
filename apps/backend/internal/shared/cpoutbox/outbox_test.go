package cpoutbox_test

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"testing"
	"testing/fstest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpoutbox"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

var migrations = fstest.MapFS{
	"1_outbox.up.sql": &fstest.MapFile{Data: []byte(`
		CREATE TYPE outbox_event AS ENUM ('google.protobuf.StringValue');
		CREATE TABLE outbox (position bigserial PRIMARY KEY, event outbox_event NOT NULL, payload bytea NOT NULL);
	`)},
	"1_outbox.down.sql": &fstest.MapFile{Data: []byte(`DROP TABLE outbox; DROP TYPE outbox_event;`)},
}

type sender struct {
	mu    sync.Mutex
	fails int
	sent  []string
}

func (s *sender) Send(_ context.Context, events ...proto.Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.fails > 0 {
		s.fails--
		return errors.New("redis is restarting")
	}
	for _, event := range events {
		s.sent = append(s.sent, event.(*wrapperspb.StringValue).GetValue()) //nolint:forcetypeassert // the outbox holds strings only.
	}
	return nil
}

func (s *sender) values() []string {
	s.mu.Lock()
	defer s.mu.Unlock()

	return append([]string(nil), s.sent...)
}

type OutboxSuite struct {
	suite.Suite
	db *cppg.Postgres
}

func TestOutbox(t *testing.T) {
	suite.Run(t, new(OutboxSuite))
}

func (s *OutboxSuite) SetupSuite() {
	s.db = cppg.StartTestServer(s.T()).OpenSchema(s.T(), "outbox", migrations)
}

func (s *OutboxSuite) SetupTest() {
	s.Require().NoError(s.db.Purge(s.T().Context()))
}

var errRolledBack = errors.New("rolled back")

func (s *OutboxSuite) write(commit bool, values ...string) {
	events := make([]proto.Message, len(values))
	for i, value := range values {
		events[i] = wrapperspb.String(value)
	}

	err := s.db.InTx(s.T().Context(), func(ctx context.Context) error {
		if err := cpoutbox.New(s.db).Record(ctx, events...); err != nil {
			return err //nolint:wrapcheck // a test.
		}
		if !commit {
			return errRolledBack
		}
		return nil
	})

	if commit {
		s.Require().NoError(err)
		return
	}
	s.Require().ErrorIs(err, errRolledBack)
}

func (s *OutboxSuite) relay(events *sender) func() {
	ctx, cancel := context.WithCancel(s.T().Context())
	done := make(chan struct{})
	go func() {
		defer close(done)
		cpoutbox.NewRelay("test-outbox", s.db, events, slog.New(slog.DiscardHandler)).Run(ctx)
	}()

	return func() {
		cancel()
		<-done
	}
}

func (s *OutboxSuite) kept() int {
	var rows int
	s.Require().NoError(s.db.On(s.T().Context()).QueryRowContext(s.T().Context(), `SELECT count(*) FROM outbox`).Scan(&rows))
	return rows
}

func (s *OutboxSuite) TestWhatACommittedWriteKeptIsSentInOrderThoughNobodyPublishedIt() {
	s.write(true, "deleted", "signed in")
	events := &sender{}

	stop := s.relay(events)
	s.Eventually(func() bool { return len(events.values()) == 2 }, 5*time.Second, 10*time.Millisecond)
	stop()

	s.Equal([]string{"deleted", "signed in"}, events.values())
	s.Zero(s.kept(), "a row sent is cleared")
}

func (s *OutboxSuite) TestARolledBackWriteSendsNothing() {
	s.write(false, "deleted")
	events := &sender{}

	s.relay(events)()

	s.Empty(events.values())
	s.Zero(s.kept())
}

func (s *OutboxSuite) TestARowRedisRefusedIsSentOnTheNextTry() {
	s.write(true, "deleted")
	events := &sender{fails: 1}

	stop := s.relay(events)
	s.Eventually(func() bool { return len(events.values()) == 1 }, 5*time.Second, 10*time.Millisecond)
	stop()

	s.Equal([]string{"deleted"}, events.values())
}

func (s *OutboxSuite) TestTheShutdownSendsWhatIsLeft() {
	events := &sender{}
	stop := s.relay(events)
	s.write(true, "deleted")

	stop()

	s.Equal([]string{"deleted"}, events.values())
}

func TestAnEventTheOutboxTypeDoesNotHoldIsRefusedWithTheWrite(t *testing.T) {
	db := cppg.StartTestServer(t).OpenSchema(t, "outbox", migrations)

	assert.Error(t, db.InTx(t.Context(), func(ctx context.Context) error {
		return cpoutbox.New(db).Record(ctx, wrapperspb.Int64(7))
	}))
}
