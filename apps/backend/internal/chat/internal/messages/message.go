package messages

import (
	"context"
	"errors"
	"time"
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
	AuthorTitle  Title
	CountryID    string
	Text         string
}

func Named(message Message, author Author) Message {
	message.AuthorName = author.Name
	message.AuthorAdmin = author.Admin
	message.AuthorColor = author.Color
	message.AuthorStreak = author.Streak
	message.AuthorTitle = author.Title
	return message
}

var ErrInvalidMessage = errors.New("invalid chat message")

type Storage interface {
	Append(ctx context.Context, record Record) error
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
