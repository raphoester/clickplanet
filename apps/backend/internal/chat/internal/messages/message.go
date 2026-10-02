package messages

import (
	"context"
	"errors"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
)

type MessageID string

type Message struct {
	ID           MessageID
	SentAt       time.Time
	Account      AccountID
	AuthorName   string
	AuthorAdmin  bool
	AuthorColor  int32
	AuthorStreak uint32
	CountryID    string
	Text         string
}

const DeletedName = "[deleted]"

func Named(message Message, authors map[AccountID]Author) Message {
	if message.Account == NoAccount {
		return message
	}

	author, known := authors[message.Account]
	if !known {
		return Message{
			ID: message.ID, SentAt: message.SentAt, Account: message.Account,
			AuthorName: DeletedName, CountryID: message.CountryID, Text: message.Text,
		}
	}

	message.AuthorName = author.Name
	message.AuthorAdmin = author.Admin
	message.AuthorColor = author.Color
	message.AuthorStreak = author.Streak
	return message
}

func AccountsOf(list []Message) []AccountID {
	seen := cpcolls.NewSetWithCapacity[AccountID](len(list))
	accounts := make([]AccountID, 0, len(list))
	for _, message := range list {
		if message.Account == NoAccount || seen.Contains(message.Account) {
			continue
		}
		seen.Add(message.Account)
		accounts = append(accounts, message.Account)
	}
	return accounts
}

var ErrInvalidMessage = errors.New("invalid chat message")

type Storage interface {
	Append(ctx context.Context, record Record) error
	Recent(ctx context.Context, since time.Time, limit int) ([]Message, error)
	Shown(ctx context.Context, id MessageID, since time.Time, limit int) (bool, error)
	DeleteBefore(ctx context.Context, cutoff time.Time) (int64, error)
}

type Window struct {
	Size      int
	Retention time.Duration
}

func (w Window) Since(now time.Time) time.Time {
	return now.Add(-w.Retention)
}
