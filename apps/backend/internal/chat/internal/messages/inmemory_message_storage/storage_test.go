package inmemory_message_storage_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages/inmemory_message_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages/postgres_message_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages/usecases/get_history_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages/usecases/listen_for_events_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages/usecases/react_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages/usecases/send_message_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(testSuite))
}

type testSuite struct {
	suite.Suite

	clock       *cptime.FixedClock
	persistence *inmemory_message_storage.MemoryPersistence
}

func (s *testSuite) SetupTest() {
	s.clock = cptime.NewFixedClock(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC))
	s.persistence = inmemory_message_storage.NewMemoryPersistence()
}

func (s *testSuite) newStorage(config inmemory_message_storage.Config) *inmemory_message_storage.Storage {
	storage := inmemory_message_storage.New(config, s.persistence, s.clock, slog.New(slog.DiscardHandler))
	s.Require().NoError(storage.Load(context.Background()))
	return storage
}

func (s *testSuite) record(text string) messages.Record {
	return messages.Record{
		Message: messages.Message{
			ID:         messages.MessageID(text),
			SentAt:     s.clock.Now(),
			AuthorName: "Bob",
			AuthorTag:  "a1b2c3",
			CountryID:  "fr",
			Text:       text,
		},
		AuthorID:  "some-uuid",
		IP:        "203.0.113.7",
		UserAgent: "test-agent",
	}
}

func (s *testSuite) texts(history []messages.Message) []string {
	texts := make([]string, 0, len(history))
	for _, message := range history {
		texts = append(texts, message.Text)
	}
	return texts
}

func (s *testSuite) TestAppendedMessagesShowUpInHistory() {
	storage := s.newStorage(inmemory_message_storage.Config{})

	s.Require().NoError(storage.Append(context.Background(), s.record("hello")))
	s.Require().NoError(storage.Append(context.Background(), s.record("planet")))

	s.Equal([]string{"hello", "planet"}, s.texts(storage.History(context.Background(), messages.NoReactor)))
}

func (s *testSuite) TestHistoryIsCapped() {
	storage := s.newStorage(inmemory_message_storage.Config{HistorySize: 3})

	for i := range 10 {
		s.Require().NoError(storage.Append(context.Background(), s.record(fmt.Sprintf("msg-%d", i))))
	}

	s.Equal([]string{"msg-7", "msg-8", "msg-9"}, s.texts(storage.History(context.Background(), messages.NoReactor)))
	s.Len(s.persistence.Stored(), 10)
}

func (s *testSuite) TestHistoryIsACopy() {
	storage := s.newStorage(inmemory_message_storage.Config{})
	s.Require().NoError(storage.Append(context.Background(), s.record("hello")))

	history := storage.History(context.Background(), messages.NoReactor)
	history[0].Text = "tampered"

	s.Equal("hello", storage.History(context.Background(), messages.NoReactor)[0].Text)
}

func (s *testSuite) TestSubscribersReceiveMessages() {
	storage := s.newStorage(inmemory_message_storage.Config{})

	feed, err := storage.Subscribe(s.T().Context())
	s.Require().NoError(err)

	s.Require().NoError(storage.Append(context.Background(), s.record("hello")))

	s.Require().Eventually(func() bool { return len(feed) == 1 }, 2*time.Second, time.Millisecond)
	s.Equal("hello", (<-feed).Message.Text)
}

func (s *testSuite) TestSubscriberChannelClosesWithItsContext() {
	storage := s.newStorage(inmemory_message_storage.Config{})

	ctx, cancel := context.WithCancel(context.Background())
	feed, err := storage.Subscribe(ctx)
	s.Require().NoError(err)

	cancel()

	s.Require().Eventually(func() bool {
		select {
		case _, open := <-feed:
			return !open
		default:
			return false
		}
	}, 2*time.Second, time.Millisecond)
}

func (s *testSuite) TestSlowSubscribersHaveMessagesDropped() {
	storage := s.newStorage(inmemory_message_storage.Config{SubscriberBuffer: 1})

	_, err := storage.Subscribe(s.T().Context())
	s.Require().NoError(err)

	for i := range 10 {
		s.Require().NoError(storage.Append(context.Background(), s.record(fmt.Sprintf("msg-%d", i))))
	}

	s.Positive(storage.DroppedMessages())
	s.Len(storage.History(context.Background(), messages.NoReactor), 10)
}

func (s *testSuite) TestTheSenderIsStoredButNotInTheHistory() {
	storage := s.newStorage(inmemory_message_storage.Config{})
	s.Require().NoError(storage.Append(context.Background(), s.record("hello")))

	s.Equal([]messages.Record{s.record("hello")}, s.persistence.Stored())

	encoded, err := json.Marshal(storage.History(context.Background(), messages.NoReactor))
	s.Require().NoError(err)
	s.NotContains(string(encoded), "203.0.113.7")
	s.NotContains(string(encoded), "test-agent")
}

func (s *testSuite) TestAMessageThatCannotBeStoredIsNotBroadcast() {
	storage := s.newStorage(inmemory_message_storage.Config{})
	feed, err := storage.Subscribe(s.T().Context())
	s.Require().NoError(err)

	s.persistence.FailWith(errors.New("postgres is down"))
	s.Require().Error(storage.Append(context.Background(), s.record("lost")))

	s.Empty(storage.History(context.Background(), messages.NoReactor))
	s.Empty(feed)

	s.persistence.Heal()
	s.Require().NoError(storage.Append(context.Background(), s.record("hello")))
	s.Equal([]string{"hello"}, s.texts(storage.History(context.Background(), messages.NoReactor)))
}

func (s *testSuite) TestLoadKeepsTheNewestMessagesWithinRetention() {
	ancient := s.record("ancient")
	s.clock.Advance(48 * time.Hour)
	s.persistence = inmemory_message_storage.NewMemoryPersistence(
		ancient, s.record("msg-1"), s.record("msg-2"), s.record("msg-3"))

	storage := s.newStorage(inmemory_message_storage.Config{HistorySize: 2, Retention: 24 * time.Hour})

	s.Equal([]string{"msg-2", "msg-3"}, s.texts(storage.History(context.Background(), messages.NoReactor)))
}

func (s *testSuite) TestAStoreThatCannotBeReadRefusesTheLoad() {
	s.persistence.FailWith(errors.New("postgres is down"))
	storage := inmemory_message_storage.New(inmemory_message_storage.Config{}, s.persistence, s.clock, slog.New(slog.DiscardHandler))

	s.Require().Error(storage.Load(context.Background()))
}

func (s *testSuite) TestRunDeletesMessagesPastRetention() {
	storage := s.newStorage(inmemory_message_storage.Config{Retention: 24 * time.Hour, PruneInterval: time.Millisecond})
	s.Require().NoError(storage.Append(context.Background(), s.record("ancient")))
	s.clock.Advance(48 * time.Hour)
	s.Require().NoError(storage.Append(context.Background(), s.record("recent")))

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		storage.Run(ctx)
	}()

	s.Require().Eventually(func() bool { return len(s.persistence.Stored()) == 1 }, 5*time.Second, time.Millisecond)
	cancel()
	<-done

	s.Equal("recent", s.persistence.Stored()[0].Message.Text)
}

func (s *testSuite) TestConcurrentUseIsSafe() {
	storage := s.newStorage(inmemory_message_storage.Config{HistorySize: 50})

	_, err := storage.Subscribe(s.T().Context())
	s.Require().NoError(err)

	// Errors are collected, not asserted in the goroutine: a failed require there would strand the WaitGroup.
	appendErrs := make(chan error, 10*20)

	var wg sync.WaitGroup
	for writer := range 10 {
		wg.Go(func() {
			for i := range 20 {
				appendErrs <- storage.Append(context.Background(), s.record(fmt.Sprintf("w%d-%d", writer, i)))
			}
		})
	}
	for range 5 {
		wg.Go(func() {
			for range 50 {
				storage.History(context.Background(), messages.NoReactor)
			}
		})
	}

	wg.Wait()
	close(appendErrs)

	for err := range appendErrs {
		s.Require().NoError(err)
	}

	s.Len(storage.History(context.Background(), messages.NoReactor), 50)
	s.Len(s.persistence.Stored(), 200)
}

var (
	_ send_message_usecase.Appender                = (*inmemory_message_storage.Storage)(nil)
	_ get_history_usecase.HistoryReader            = (*inmemory_message_storage.Storage)(nil)
	_ react_usecase.Board                          = (*inmemory_message_storage.Storage)(nil)
	_ listen_for_events_usecase.MessagesSubscriber = (*inmemory_message_storage.Storage)(nil)
	_ inmemory_message_storage.Persistence         = (*postgres_message_store.Store)(nil)
	_ inmemory_message_storage.Persistence         = (*inmemory_message_storage.MemoryPersistence)(nil)
)
