// Package messages is the chat's messages: a message, the rules it passes, and where it is kept.
package messages

import (
	"context"
	"errors"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
)

// MessageID names a message. The server makes it, a uuid, when it accepts the message.
type MessageID string

type Message struct {
	ID     MessageID
	SentAt time.Time
	// Account is who sent it. It is the message's link to its player, and what the name below is read from:
	// NoAccount only on a message written before the chat kept it.
	Account AccountID
	// AuthorName is who the message is from, as a reader sees it. Named fills it from who the account is now,
	// so a rename shows on everything its player ever said; on a message with no account it is the copy the
	// row carries, which is all such a message has.
	AuthorName string
	// AuthorAdmin is whether that author is an admin of the game. Never a guest. Filled with AuthorName.
	AuthorAdmin bool
	CountryID   string
	Text        string
}

// DeletedName stands in for an account that can no longer be named: it was deleted. Showing the name it used
// would undo the deletion, so nothing does. No username can look like it — SetName refuses punctuation — and no
// guest code can either.
const DeletedName = "[deleted]"

// Named is the message as a reader sees it now, given who the accounts in the window are.
//
// A message with an account takes that account's name, or DeletedName when nobody can name it any more. One
// with no account is from before the chat kept them and keeps the copy in its row. Its admin mark follows the
// same rule, so a player that stops being an admin stops looking like one everywhere.
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
	return message
}

// AccountsOf is every account the messages are from, each once, for one batch ask. A message with none is left
// out: nobody has to be asked about it.
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
