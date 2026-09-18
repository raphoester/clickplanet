// Package messages is the chat's messages: a message, the rules it passes, and where it is kept.
package messages

import (
	"context"
	"errors"
	"time"
)

// MessageID names a message. The server makes it, a uuid, when it accepts the message.
type MessageID string

type Message struct {
	ID     MessageID
	SentAt time.Time
	// AuthorName is a username, or "guest_" and the account's guest code, as the player module named it.
	AuthorName string
	// AuthorAdmin is whether the username it was sent under was an admin's then. Never a guest.
	AuthorAdmin bool
	CountryID   string
	Text        string
}

var ErrInvalidMessage = errors.New("invalid chat message")

// Storage is where messages are kept. StorageContractSuite pins what every adapter does.
type Storage interface {
	// Append keeps a message; the log is the audit trail, so a message that cannot be kept is not sent.
	Append(ctx context.Context, record Record) error
	// Recent is the newest limit messages sent at or after since, in the order they were appended.
	Recent(ctx context.Context, since time.Time, limit int) ([]Message, error)
	// Shown is whether the message is one of those Recent answers.
	Shown(ctx context.Context, id MessageID, since time.Time, limit int) (bool, error)
	// DeleteBefore removes every message sent before cutoff and says how many.
	DeleteBefore(ctx context.Context, cutoff time.Time) (int64, error)
}

// Window is which messages the chat shows: the newest Size, sent within Retention.
type Window struct {
	Size      int
	Retention time.Duration
}

// Since is the oldest a shown message may be at now.
func (w Window) Since(now time.Time) time.Time {
	return now.Add(-w.Retention)
}
