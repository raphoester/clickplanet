package inmemory_message_storage

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

func New(
	config Config,
	persistence Persistence,
	clock cptime.Clock,
	logger *slog.Logger,
) *Storage {
	config = config.withDefaults()

	return &Storage{
		config:      config,
		persistence: persistence,
		logger:      logger,
		clock:       clock,
		history:     make([]messages.Message, 0, config.HistorySize),
		subscribers: make(map[*subscriber]struct{}),
	}
}

type Storage struct {
	config      Config
	persistence Persistence
	logger      *slog.Logger
	clock       cptime.Clock

	appendMu sync.Mutex

	historyMu sync.RWMutex
	history   []messages.Message

	subscribersMu sync.Mutex
	subscribers   map[*subscriber]struct{}
}

type subscriber struct {
	ch      chan messages.Event
	dropped atomic.Uint64
}

const writeTimeout = 5 * time.Second

// Append records the message before anyone sees it: a message that cannot be recorded is not broadcast.
func (s *Storage) Append(ctx context.Context, record messages.Record) error {
	s.appendMu.Lock()
	defer s.appendMu.Unlock()

	ctx, cancel := context.WithTimeout(ctx, writeTimeout)
	defer cancel()

	if err := s.persistence.Insert(ctx, record); err != nil {
		return fmt.Errorf("failed to record the chat message: %w", err)
	}

	s.remember(record.Message)
	s.publish(messages.Event{Message: &record.Message})

	return nil
}

// Redact blanks one author on every open screen and counts what it hid. The
// history keeps its text: the ban is read over it on the way out instead.
func (s *Storage) Redact(tag string) int {
	s.publish(messages.Event{Redaction: &messages.Redaction{AuthorTag: tag}})

	s.historyMu.RLock()
	defer s.historyMu.RUnlock()

	redacted := 0
	for _, message := range s.history {
		if message.AuthorTag == tag {
			redacted++
		}
	}

	return redacted
}

func (s *Storage) History(_ context.Context) []messages.Message {
	s.historyMu.RLock()
	defer s.historyMu.RUnlock()

	return append(make([]messages.Message, 0, len(s.history)), s.history...)
}

func (s *Storage) remember(message messages.Message) {
	s.historyMu.Lock()
	defer s.historyMu.Unlock()

	if len(s.history) == s.config.HistorySize {
		s.history = append(s.history[:0], s.history[1:]...)
	}

	s.history = append(s.history, message)
}

func (s *Storage) Subscribe(ctx context.Context) (<-chan messages.Event, error) {
	sub := &subscriber{ch: make(chan messages.Event, s.config.SubscriberBuffer)}

	s.subscribersMu.Lock()
	s.subscribers[sub] = struct{}{}
	s.subscribersMu.Unlock()

	go func() {
		<-ctx.Done()

		s.subscribersMu.Lock()
		defer s.subscribersMu.Unlock()

		delete(s.subscribers, sub)
		close(sub.ch)
	}()

	return sub.ch, nil
}

const dropLogInterval = 100

func (s *Storage) publish(event messages.Event) {
	s.subscribersMu.Lock()
	defer s.subscribersMu.Unlock()

	for sub := range s.subscribers {
		select {
		case sub.ch <- event:
		default:
			dropped := sub.dropped.Add(1)
			if dropped == 1 || dropped%dropLogInterval == 0 {
				s.logger.Warn("dropped a chat event for a slow subscriber",
					slog.String("event", describe(event)),
					slog.Uint64("droppedTotal", dropped),
				)
			}
		}
	}
}

// describe names the dropped event, so a redaction a client never got is not read as a lost message.
func describe(event messages.Event) string {
	if event.Redaction != nil {
		return "redaction of #" + event.Redaction.AuthorTag
	}
	return "message " + event.Message.ID
}

func (s *Storage) DroppedMessages() uint64 {
	s.subscribersMu.Lock()
	defer s.subscribersMu.Unlock()

	var total uint64
	for sub := range s.subscribers {
		total += sub.dropped.Load()
	}
	return total
}
