package inmemory_message_storage

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
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
		reactions:   make(map[messages.MessageID]messages.Reactions),
		subscribers: cpcolls.NewSet[*subscriber](),
	}
}

type Storage struct {
	config      Config
	persistence Persistence
	logger      *slog.Logger
	clock       cptime.Clock

	// writeMu orders every write, a message or a reaction, so the stream sends them in the order they were
	// recorded, and never a message's reactions before the message.
	writeMu sync.Mutex

	historyMu sync.RWMutex
	history   []messages.Message
	// reactions holds what a message in history carries; a message leaves it with the history.
	reactions map[messages.MessageID]messages.Reactions

	subscribersMu sync.Mutex
	subscribers   *cpcolls.Set[*subscriber]
}

type subscriber struct {
	ch      chan messages.Update
	dropped atomic.Uint64
}

const writeTimeout = 5 * time.Second

// Append records the message before anyone sees it: a message that cannot be recorded is not broadcast.
func (s *Storage) Append(ctx context.Context, record messages.Record) error {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()

	ctx, cancel := context.WithTimeout(ctx, writeTimeout)
	defer cancel()

	if err := s.persistence.Insert(ctx, record); err != nil {
		return fmt.Errorf("failed to record the chat message: %w", err)
	}

	message := record.Message
	message.Reactions = nil
	s.remember(message)
	s.publish(messages.Update{Message: &message})

	return nil
}

// React records a change to a message's reactions, then publishes all of them, and answers them. A change that
// changes nothing is neither recorded nor published. Only a message in history can be reacted to: nobody is
// shown any other.
func (s *Storage) React(ctx context.Context, change messages.ReactionChange) (messages.Reactions, error) {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()

	current, shown := s.reactionsOf(change.MessageID)
	if !shown {
		return messages.Reactions{}, fmt.Errorf("%w: %q", messages.ErrUnknownMessage, change.MessageID)
	}

	if current.Given(change.Reaction, change.Reactor) == change.On {
		return current, nil
	}

	ctx, cancel := context.WithTimeout(ctx, writeTimeout)
	defer cancel()

	if err := s.record(ctx, change); err != nil {
		return messages.Reactions{}, err
	}

	next := current.Applied(change)

	s.historyMu.Lock()
	s.reactions[change.MessageID] = next
	s.historyMu.Unlock()

	s.publish(messages.Update{Reactions: &messages.Tally{
		MessageID: change.MessageID,
		Counts:    next.Tally(messages.NoReactor),
	}})

	return next, nil
}

func (s *Storage) record(ctx context.Context, change messages.ReactionChange) error {
	if change.On {
		if err := s.persistence.InsertReaction(ctx, change); err != nil {
			return fmt.Errorf("failed to record the reaction: %w", err)
		}
		return nil
	}

	if err := s.persistence.DeleteReaction(ctx, change); err != nil {
		return fmt.Errorf("failed to take the reaction off: %w", err)
	}
	return nil
}

// reactionsOf is what a message carries, and whether it is in history at all.
func (s *Storage) reactionsOf(id messages.MessageID) (messages.Reactions, bool) {
	s.historyMu.RLock()
	defer s.historyMu.RUnlock()

	if !slices.ContainsFunc(s.history, func(message messages.Message) bool { return message.ID == id }) {
		return messages.Reactions{}, false
	}
	return s.reactions[id], true
}

// History is the recent messages, each with its reactions as viewer sees them.
func (s *Storage) History(_ context.Context, viewer messages.Reactor) []messages.Message {
	s.historyMu.RLock()
	defer s.historyMu.RUnlock()

	history := make([]messages.Message, 0, len(s.history))
	for _, message := range s.history {
		message.Reactions = s.reactions[message.ID].Tally(viewer)
		history = append(history, message)
	}
	return history
}

func (s *Storage) remember(message messages.Message) {
	s.historyMu.Lock()
	defer s.historyMu.Unlock()

	if len(s.history) == s.config.HistorySize {
		delete(s.reactions, s.history[0].ID)
		s.history = append(s.history[:0], s.history[1:]...)
	}

	s.history = append(s.history, message)
}

func (s *Storage) Subscribe(ctx context.Context) (<-chan messages.Update, error) {
	sub := &subscriber{ch: make(chan messages.Update, s.config.SubscriberBuffer)}

	s.subscribersMu.Lock()
	s.subscribers.Add(sub)
	s.subscribersMu.Unlock()

	go func() {
		<-ctx.Done()

		s.subscribersMu.Lock()
		defer s.subscribersMu.Unlock()

		s.subscribers.Delete(sub)
		close(sub.ch)
	}()

	return sub.ch, nil
}

const dropLogInterval = 100

func (s *Storage) publish(update messages.Update) {
	s.subscribersMu.Lock()
	defer s.subscribersMu.Unlock()

	s.subscribers.ForEach(func(sub *subscriber) {
		select {
		case sub.ch <- update:
		default:
			dropped := sub.dropped.Add(1)
			if dropped == 1 || dropped%dropLogInterval == 0 {
				s.logger.Warn("dropped a chat update for a slow subscriber",
					slog.String("messageId", string(update.MessageID())),
					slog.Uint64("droppedTotal", dropped),
				)
			}
		}
	})
}

func (s *Storage) DroppedMessages() uint64 {
	s.subscribersMu.Lock()
	defer s.subscribersMu.Unlock()

	var total uint64
	s.subscribers.ForEach(func(sub *subscriber) {
		total += sub.dropped.Load()
	})
	return total
}
