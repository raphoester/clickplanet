// Package get_history_usecase reads the recent messages a joining client is shown, each with its reactions.
package get_history_usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/reactions"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type MessageReader interface {
	Recent(ctx context.Context, since time.Time, limit int) ([]messages.Message, error)
}

type ReactionReader interface {
	Reactions(ctx context.Context, ids []messages.MessageID) (map[messages.MessageID]reactions.Reactions, error)
}

// Entry is one message of the history, with its reactions as the caller sees them, and their version.
type Entry struct {
	Message          messages.Message
	Reactions        []reactions.Count
	ReactionsVersion uint64
}

func New(
	messageReader MessageReader,
	reactionReader ReactionReader,
	clock cptime.Clock,
	window messages.Window,
) *UseCase {
	return &UseCase{
		messages:  messageReader,
		reactions: reactionReader,
		clock:     clock,
		window:    window,
	}
}

type UseCase struct {
	messages  MessageReader
	reactions ReactionReader
	clock     cptime.Clock
	window    messages.Window
}

// Execute marks the caller's own reactions: those of its account. A caller with none has none.
func (u *UseCase) Execute(ctx context.Context, account messages.AccountID) ([]Entry, error) {
	recent, err := u.messages.Recent(ctx, u.window.Since(u.clock.Now()), u.window.Size)
	if err != nil {
		return nil, fmt.Errorf("failed to read the chat history: %w", err)
	}

	ids := make([]messages.MessageID, 0, len(recent))
	for _, message := range recent {
		ids = append(ids, message.ID)
	}
	given, err := u.reactions.Reactions(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("failed to read the chat reactions: %w", err)
	}

	viewer := reactions.ReactorOf(account)

	history := make([]Entry, 0, len(recent))
	for _, message := range recent {
		history = append(history, Entry{
			Message:          message,
			Reactions:        given[message.ID].Tally(viewer),
			ReactionsVersion: given[message.ID].Version(),
		})
	}
	return history, nil
}
