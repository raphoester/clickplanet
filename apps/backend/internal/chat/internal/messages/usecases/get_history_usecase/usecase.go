package get_history_usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/announcements"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/reactions"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
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

type Authors interface {
	Authors(ctx context.Context, accounts []messages.AccountID) (map[messages.AccountID]messages.Author, error)
}

type Entry struct {
	Message          messages.Message
	Reactions        []reactions.Count
	ReactionsVersion uint64
}

type History struct {
	Messages      []Entry
	Announcements []announcements.Announcement
	SeenUntil     time.Time
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

func (u *UseCase) Execute(ctx context.Context, account messages.AccountID) (History, error) {
	now := u.clock.Now()

	recent, err := u.messages.Recent(ctx, u.window.Since(now), u.window.Size)
	if err != nil {
		return History{}, fmt.Errorf("failed to read the chat history: %w", err)
	}

	announced, err := u.announcements.Recent(ctx, u.window.Beginning(now, recent), u.window.Size)
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

	viewer := reactions.ReactorOf(account)

	tallies := make(map[messages.MessageID][]reactions.Count, len(recent))
	for _, message := range recent {
		tallies[message.ID] = given[message.ID].Tally(viewer)
	}

	named, err := u.authors.Authors(ctx, everyone(recent, tallies))
	if err != nil {
		return History{}, fmt.Errorf("failed to read who the chat history is from: %w", err)
	}

	history := make([]Entry, 0, len(recent))
	for _, message := range recent {
		history = append(history, Entry{
			Message:          messages.Named(message, named),
			Reactions:        reactions.Named(tallies[message.ID], named),
			ReactionsVersion: given[message.ID].Version(),
		})
	}
	return History{Messages: history, Announcements: announced}, nil
}

func everyone(recent []messages.Message, tallies map[messages.MessageID][]reactions.Count) []messages.AccountID {
	accounts := messages.AccountsOf(recent)
	seen := cpcolls.NewSet(accounts...)

	for _, message := range recent {
		for _, account := range reactions.AccountsOf(tallies[message.ID]) {
			if seen.Contains(account) {
				continue
			}
			seen.Add(account)
			accounts = append(accounts, account)
		}
	}
	return accounts
}
