// Package get_history_usecase reads the recent messages a joining client is shown, each with its reactions.
package get_history_usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/reactions"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type MessageReader interface {
	Recent(ctx context.Context, since time.Time, limit int) ([]messages.Message, error)
}

type ReactionReader interface {
	Reactions(ctx context.Context, ids []messages.MessageID) (map[messages.MessageID]reactions.Reactions, error)
}

// Authors is the player module, asked who reads, so the caller's own reactions can say so.
type Authors interface {
	Author(ctx context.Context, account messages.AccountID, ip string) (messages.Author, error)
}

// Entry is one message of the history, with its reactions as the caller sees them.
type Entry struct {
	Message   messages.Message
	Reactions []reactions.Count
}

func New(
	messageReader MessageReader,
	reactionReader ReactionReader,
	authors Authors,
	clock cptime.Clock,
	window messages.Window,
) *UseCase {
	return &UseCase{
		messages:  messageReader,
		reactions: reactionReader,
		authors:   authors,
		clock:     clock,
		window:    window,
	}
}

type UseCase struct {
	messages  MessageReader
	reactions ReactionReader
	authors   Authors
	clock     cptime.Clock
	window    messages.Window
}

// Execute serves the history even when the player module does not answer: nothing is then marked as the caller's.
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

	viewer := reactions.NoReactor
	if author, err := u.authors.Author(ctx, account, cpctx.GetSourceIP(ctx)); err == nil {
		viewer = reactions.ReactorOf(account, author)
	}

	history := make([]Entry, 0, len(recent))
	for _, message := range recent {
		history = append(history, Entry{Message: message, Reactions: given[message.ID].Tally(viewer)})
	}
	return history, nil
}
