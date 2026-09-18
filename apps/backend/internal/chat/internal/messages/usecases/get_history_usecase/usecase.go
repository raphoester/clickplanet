// Package get_history_usecase reads the recent messages a joining client is shown, each with its reactions, and
// the announcements between them.
package get_history_usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/announcements"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/reactions"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type MessageReader interface {
	Recent(ctx context.Context, since time.Time, limit int) ([]messages.Message, error)
}

type AnnouncementReader interface {
	Recent(ctx context.Context, since time.Time, limit int) ([]announcements.Announcement, error)
}

type ReactionReader interface {
	Reactions(ctx context.Context, ids []messages.MessageID) (map[messages.MessageID]reactions.Reactions, error)
}

// Authors is the player module, asked who reads, so the caller's own reactions can say so.
type Authors interface {
	Author(ctx context.Context, account messages.AccountID, ip string) (messages.Author, error)
}

// Entry is one message of the history, with its reactions as the caller sees them, and their version.
type Entry struct {
	Message          messages.Message
	Reactions        []reactions.Count
	ReactionsVersion uint64
}

// History is what a joining client is shown: the messages and the announcements, each oldest first. The window
// bounds each on its own, so a burst of bombs never pushes the messages out.
type History struct {
	Messages      []Entry
	Announcements []announcements.Announcement
}

func New(
	messageReader MessageReader,
	reactionReader ReactionReader,
	announcementReader AnnouncementReader,
	authors Authors,
	clock cptime.Clock,
	window messages.Window,
) *UseCase {
	return &UseCase{
		messages:      messageReader,
		reactions:     reactionReader,
		announcements: announcementReader,
		authors:       authors,
		clock:         clock,
		window:        window,
	}
}

type UseCase struct {
	messages      MessageReader
	reactions     ReactionReader
	announcements AnnouncementReader
	authors       Authors
	clock         cptime.Clock
	window        messages.Window
}

// Execute serves the history even when the player module does not answer: nothing is then marked as the caller's.
func (u *UseCase) Execute(ctx context.Context, account messages.AccountID) (History, error) {
	since := u.window.Since(u.clock.Now())

	recent, err := u.messages.Recent(ctx, since, u.window.Size)
	if err != nil {
		return History{}, fmt.Errorf("failed to read the chat history: %w", err)
	}

	announced, err := u.announcements.Recent(ctx, since, u.window.Size)
	if err != nil {
		return History{}, fmt.Errorf("failed to read the chat announcements: %w", err)
	}

	ids := make([]messages.MessageID, 0, len(recent))
	for _, message := range recent {
		ids = append(ids, message.ID)
	}
	given, err := u.reactions.Reactions(ctx, ids)
	if err != nil {
		return History{}, fmt.Errorf("failed to read the chat reactions: %w", err)
	}

	viewer := reactions.NoReactor
	if author, err := u.authors.Author(ctx, account, cpctx.GetSourceIP(ctx)); err == nil {
		viewer = reactions.ReactorOf(account, author)
	}

	history := make([]Entry, 0, len(recent))
	for _, message := range recent {
		history = append(history, Entry{
			Message:          message,
			Reactions:        given[message.ID].Tally(viewer),
			ReactionsVersion: given[message.ID].Version(),
		})
	}
	return History{Messages: history, Announcements: announced}, nil
}
