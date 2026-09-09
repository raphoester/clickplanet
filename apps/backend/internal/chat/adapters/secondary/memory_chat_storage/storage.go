package memory_chat_storage

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/domain"
	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/logging"
	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/logging/lf"
	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/xtime"
)

func New(
	config Config,
	timeProvider xtime.Provider,
	logger logging.Logger,
) *Storage {
	if logger == nil {
		logger = logging.NewNopLogger()
	}
	if timeProvider == nil {
		timeProvider = xtime.ActualProvider{}
	}

	config = config.withDefaults()

	s := &Storage{
		config:       config,
		logger:       logger,
		timeProvider: timeProvider,
		history:      make([]domain.ChatMessage, 0, config.HistorySize),
		subscribers:  make(map[*subscriber]struct{}),
	}

	s.restore()

	return s
}

type Storage struct {
	config       Config
	logger       logging.Logger
	timeProvider xtime.Provider

	historyMu sync.RWMutex
	history   []domain.ChatMessage

	subscribersMu sync.Mutex
	subscribers   map[*subscriber]struct{}

	logMu sync.Mutex
	log   *appendLog
}

type subscriber struct {
	ch      chan domain.ChatMessage
	dropped atomic.Uint64
}

func (s *Storage) Append(_ context.Context, record domain.ChatRecord) error {
	if err := s.appendToLog(record); err != nil {
		return fmt.Errorf("failed to write to the chat log: %w", err)
	}

	s.remember(record.Message)
	s.publish(record.Message)

	return nil
}

func (s *Storage) History(_ context.Context) []domain.ChatMessage {
	s.historyMu.RLock()
	defer s.historyMu.RUnlock()

	return append(make([]domain.ChatMessage, 0, len(s.history)), s.history...)
}

func (s *Storage) remember(message domain.ChatMessage) {
	s.historyMu.Lock()
	defer s.historyMu.Unlock()

	if len(s.history) == s.config.HistorySize {
		s.history = append(s.history[:0], s.history[1:]...)
	}

	s.history = append(s.history, message)
}

func (s *Storage) Subscribe(ctx context.Context) (<-chan domain.ChatMessage, error) {
	sub := &subscriber{ch: make(chan domain.ChatMessage, s.config.SubscriberBuffer)}

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

func (s *Storage) publish(message domain.ChatMessage) {
	s.subscribersMu.Lock()
	defer s.subscribersMu.Unlock()

	for sub := range s.subscribers {
		select {
		case sub.ch <- message:
		default:
			dropped := sub.dropped.Add(1)
			if dropped == 1 || dropped%dropLogInterval == 0 {
				s.logger.Warning("dropped a chat message for a slow subscriber",
					lf.String("messageId", message.ID),
					lf.Any("droppedTotal", dropped),
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
