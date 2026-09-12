package memory_chat_storage_test

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/adapters/secondary/memory_chat_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/domain"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
	"github.com/stretchr/testify/suite"
)

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(testSuite))
}

type testSuite struct {
	suite.Suite

	clock   *fakeClock
	logPath string
}

func (s *testSuite) SetupTest() {
	s.clock = &fakeClock{now: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}
	s.logPath = filepath.Join(s.T().TempDir(), "chat.log")
}

func (s *testSuite) newStorage(config memory_chat_storage.Config) *memory_chat_storage.Storage {
	config.LogPath = s.logPath
	return memory_chat_storage.New(config, s.clock, slog.New(slog.DiscardHandler))
}

func (s *testSuite) record(text string) domain.ChatRecord {
	return domain.ChatRecord{
		Message: domain.ChatMessage{
			ID:         text,
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

func (s *testSuite) readLog() string {
	content, err := os.ReadFile(s.logPath)
	s.Require().NoError(err)
	return string(content)
}

func (s *testSuite) TestAppendedMessagesShowUpInHistory() {
	storage := s.newStorage(memory_chat_storage.Config{})

	s.Require().NoError(storage.Append(context.Background(), s.record("hello")))
	s.Require().NoError(storage.Append(context.Background(), s.record("planet")))

	history := storage.History(context.Background())
	s.Require().Len(history, 2)
	s.Equal("hello", history[0].Text)
	s.Equal("planet", history[1].Text)
}

func (s *testSuite) TestHistoryIsCapped() {
	storage := s.newStorage(memory_chat_storage.Config{HistorySize: 3})

	for i := range 10 {
		s.Require().NoError(storage.Append(context.Background(), s.record(fmt.Sprintf("msg-%d", i))))
	}

	history := storage.History(context.Background())
	s.Require().Len(history, 3)
	s.Equal("msg-7", history[0].Text)
	s.Equal("msg-9", history[2].Text)
}

func (s *testSuite) TestHistoryIsACopy() {
	storage := s.newStorage(memory_chat_storage.Config{})
	s.Require().NoError(storage.Append(context.Background(), s.record("hello")))

	history := storage.History(context.Background())
	history[0].Text = "tampered"

	s.Equal("hello", storage.History(context.Background())[0].Text)
}

func (s *testSuite) TestSubscribersReceiveMessages() {
	storage := s.newStorage(memory_chat_storage.Config{})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	messages, err := storage.Subscribe(ctx)
	s.Require().NoError(err)

	s.Require().NoError(storage.Append(context.Background(), s.record("hello")))

	select {
	case message := <-messages:
		s.Equal("hello", message.Text)
	case <-time.After(2 * time.Second):
		s.T().Fatal("the subscriber never received the message")
	}
}

func (s *testSuite) TestSubscriberChannelClosesWithItsContext() {
	storage := s.newStorage(memory_chat_storage.Config{})

	ctx, cancel := context.WithCancel(context.Background())
	messages, err := storage.Subscribe(ctx)
	s.Require().NoError(err)

	cancel()

	select {
	case _, open := <-messages:
		s.False(open)
	case <-time.After(2 * time.Second):
		s.T().Fatal("the subscriber channel was never closed")
	}
}

func (s *testSuite) TestSlowSubscribersHaveMessagesDropped() {
	storage := s.newStorage(memory_chat_storage.Config{SubscriberBuffer: 1})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	_, err := storage.Subscribe(ctx)
	s.Require().NoError(err)

	for i := range 10 {
		s.Require().NoError(storage.Append(context.Background(), s.record(fmt.Sprintf("msg-%d", i))))
	}

	s.Positive(storage.DroppedMessages())
	s.Len(storage.History(context.Background()), 10)
}

func (s *testSuite) TestTheSenderIPReachesTheLogButNotTheHistory() {
	storage := s.newStorage(memory_chat_storage.Config{})
	s.Require().NoError(storage.Append(context.Background(), s.record("hello")))

	s.Contains(s.readLog(), "203.0.113.7")

	encoded, err := json.Marshal(storage.History(context.Background()))
	s.Require().NoError(err)
	s.NotContains(string(encoded), "203.0.113.7")
	s.NotContains(string(encoded), "test-agent")
}

func (s *testSuite) TestOneLinePerMessage() {
	storage := s.newStorage(memory_chat_storage.Config{})

	for i := range 3 {
		s.Require().NoError(storage.Append(context.Background(), s.record(fmt.Sprintf("msg-%d", i))))
	}

	s.Len(strings.Split(strings.TrimSpace(s.readLog()), "\n"), 3)
}

func (s *testSuite) TestHistorySurvivesARestart() {
	storage := s.newStorage(memory_chat_storage.Config{})
	s.Require().NoError(storage.Append(context.Background(), s.record("hello")))
	storage.Run(cancelledContext())

	restarted := s.newStorage(memory_chat_storage.Config{})

	history := restarted.History(context.Background())
	s.Require().Len(history, 1)
	s.Equal("hello", history[0].Text)
	s.Equal("a1b2c3", history[0].AuthorTag)
}

func (s *testSuite) TestRestoreKeepsOnlyTheMostRecentHistory() {
	storage := s.newStorage(memory_chat_storage.Config{})
	for i := range 10 {
		s.Require().NoError(storage.Append(context.Background(), s.record(fmt.Sprintf("msg-%d", i))))
	}
	storage.Run(cancelledContext())

	restarted := s.newStorage(memory_chat_storage.Config{HistorySize: 3})

	history := restarted.History(context.Background())
	s.Require().Len(history, 3)
	s.Equal("msg-9", history[2].Text)
}

func (s *testSuite) TestRestoreIgnoresMessagesPastRetention() {
	storage := s.newStorage(memory_chat_storage.Config{})
	s.Require().NoError(storage.Append(context.Background(), s.record("ancient")))

	s.clock.advance(48 * time.Hour)
	s.Require().NoError(storage.Append(context.Background(), s.record("recent")))
	storage.Run(cancelledContext())

	restarted := s.newStorage(memory_chat_storage.Config{Retention: 24 * time.Hour})

	history := restarted.History(context.Background())
	s.Require().Len(history, 1)
	s.Equal("recent", history[0].Text)
}

func (s *testSuite) TestACorruptLineCostsHistoryNotTheStart() {
	s.Require().NoError(os.WriteFile(s.logPath, []byte("{not json\n"), 0o600))

	storage := s.newStorage(memory_chat_storage.Config{})

	s.Empty(storage.History(context.Background()))
	s.Require().NoError(storage.Append(context.Background(), s.record("hello")))
	s.Len(storage.History(context.Background()), 1)
}

func (s *testSuite) TestAppendingIsAppendingNotOverwriting() {
	first := s.newStorage(memory_chat_storage.Config{})
	s.Require().NoError(first.Append(context.Background(), s.record("hello")))
	first.Run(cancelledContext())

	second := s.newStorage(memory_chat_storage.Config{})
	s.Require().NoError(second.Append(context.Background(), s.record("planet")))
	second.Run(cancelledContext())

	log := s.readLog()
	s.Contains(log, "hello")
	s.Contains(log, "planet")
}

func (s *testSuite) TestPruningDropsExpiredRecords() {
	storage := s.newStorage(memory_chat_storage.Config{
		Retention:     24 * time.Hour,
		PruneInterval: time.Millisecond,
		FlushInterval: time.Hour,
	})

	s.Require().NoError(storage.Append(context.Background(), s.record("ancient")))
	s.clock.advance(48 * time.Hour)
	s.Require().NoError(storage.Append(context.Background(), s.record("recent")))

	stop := s.startRunning(storage)
	s.waitUntilGone("ancient")
	stop()

	log := s.readLog()
	s.NotContains(log, "ancient")
	s.Contains(log, "recent")
}

func (s *testSuite) TestAppendingStillWorksAfterAPrune() {
	storage := s.newStorage(memory_chat_storage.Config{
		Retention:     24 * time.Hour,
		PruneInterval: time.Millisecond,
		FlushInterval: time.Millisecond,
	})

	s.Require().NoError(storage.Append(context.Background(), s.record("ancient")))
	s.clock.advance(48 * time.Hour)

	stop := s.startRunning(storage)
	s.waitUntilGone("ancient")

	s.Require().NoError(storage.Append(context.Background(), s.record("after-prune")))
	stop()

	restarted := s.newStorage(memory_chat_storage.Config{Retention: 24 * time.Hour})
	history := restarted.History(context.Background())
	s.Require().Len(history, 1)
	s.Equal("after-prune", history[0].Text)
}

func (s *testSuite) startRunning(storage *memory_chat_storage.Storage) func() {
	s.T().Helper()

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		storage.Run(ctx)
	}()

	return func() {
		cancel()
		<-done
	}
}

func (s *testSuite) waitUntilGone(text string) {
	s.T().Helper()

	s.Require().Eventually(func() bool {
		content, err := os.ReadFile(s.logPath)
		return err == nil && !strings.Contains(string(content), text)
	}, 5*time.Second, 5*time.Millisecond)
}

func (s *testSuite) TestNoLogPathKeepsChatInMemory() {
	storage := memory_chat_storage.New(memory_chat_storage.Config{}, s.clock, slog.New(slog.DiscardHandler))

	s.Require().NoError(storage.Append(context.Background(), s.record("hello")))
	s.Len(storage.History(context.Background()), 1)

	storage.Run(cancelledContext())
	s.NoFileExists(s.logPath)
}

func (s *testSuite) TestConcurrentUseIsSafe() {
	storage := s.newStorage(memory_chat_storage.Config{HistorySize: 50})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	_, err := storage.Subscribe(ctx)
	s.Require().NoError(err)

	// Append errors are collected rather than asserted in the goroutine: a failed
	// require there calls runtime.Goexit on that goroutine, which would strand the
	// WaitGroup and let the suite report a pass.
	appendErrs := make(chan error, 10*20)

	var wg sync.WaitGroup
	for writer := range 10 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range 20 {
				if err := storage.Append(context.Background(), s.record(fmt.Sprintf("w%d-%d", writer, i))); err != nil {
					appendErrs <- err
					return
				}
			}
		}()
	}

	for range 5 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 50 {
				storage.History(context.Background())
			}
		}()
	}

	wg.Wait()
	close(appendErrs)

	for err := range appendErrs {
		s.Require().NoError(err)
	}

	s.Len(storage.History(context.Background()), 50)
	s.Len(strings.Split(strings.TrimSpace(s.readLog()), "\n"), 200)
}

func cancelledContext() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	return ctx
}

type fakeClock struct {
	mu  sync.Mutex
	now time.Time
}

func (c *fakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *fakeClock) advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

var (
	_ cptime.Provider = (*fakeClock)(nil)
	_ domain.Storage  = (*memory_chat_storage.Storage)(nil)
)
