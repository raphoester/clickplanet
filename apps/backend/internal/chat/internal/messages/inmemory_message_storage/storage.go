package inmemory_message_storage

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

func New(
	config Config,
	clock cptime.Clock,
	logger *slog.Logger,
) *Storage {
	if logger == nil {
		logger = slog.New(slog.DiscardHandler)
	}
	if clock == nil {
		clock = cptime.SystemClock{}
	}

	config = config.withDefaults()

	s := &Storage{
		config:      config,
		logger:      logger,
		clock:       clock,
		history:     make([]messages.Message, 0, config.HistorySize),
		subscribers: make(map[*subscriber]struct{}),
	}

	return s
}

type Storage struct {
	config Config
	logger *slog.Logger
	clock  cptime.Clock

	historyMu sync.RWMutex
	history   []messages.Message

	subscribersMu sync.Mutex
	subscribers   map[*subscriber]struct{}

	logMu sync.Mutex
	log   *appendLog
}

type subscriber struct {
	ch      chan messages.Message
	dropped atomic.Uint64
}

func (s *Storage) Append(_ context.Context, record messages.Record) error {
	if err := s.appendToLog(record); err != nil {
		return fmt.Errorf("failed to write to the chat log: %w", err)
	}

	s.remember(record.Message)
	s.publish(record.Message)

	return nil
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

func (s *Storage) Subscribe(ctx context.Context) (<-chan messages.Message, error) {
	sub := &subscriber{ch: make(chan messages.Message, s.config.SubscriberBuffer)}

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

func (s *Storage) publish(message messages.Message) {
	s.subscribersMu.Lock()
	defer s.subscribersMu.Unlock()

	for sub := range s.subscribers {
		select {
		case sub.ch <- message:
		default:
			dropped := sub.dropped.Add(1)
			if dropped == 1 || dropped%dropLogInterval == 0 {
				s.logger.Warn("dropped a chat message for a slow subscriber",
					slog.String("messageId", message.ID),
					slog.Uint64("droppedTotal", dropped),
				)
			}
		}
	}
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
