// Package messages is the chat's one concept: a message, and the rules it passes.
package messages

import (
	"errors"
	"time"
)

// MessageID names a message. The server makes it, a uuid, when it accepts the message.
type MessageID string

type Message struct {
	ID         MessageID
	SentAt     time.Time
	AuthorName string
	AuthorTag  string
	// AuthorAdmin is whether the username it was sent under was an admin's then. Never a guest.
	AuthorAdmin bool
	CountryID   string
	Text        string
	// Reactions is the message's reactions as its reader sees them. Filled when it is read, never stored.
	Reactions []Count
}

// Update is one frame of the chat's live feed: exactly one of a message sent and a message's new reactions.
type Update struct {
	Message   *Message
	Reactions *Tally
}

var ErrInvalidMessage = errors.New("invalid chat message")

// MessageID is the message the update is about, whichever kind it is.
func (u Update) MessageID() MessageID {
	if u.Reactions != nil {
		return u.Reactions.MessageID
	}
	if u.Message != nil {
		return u.Message.ID
	}
	return ""
}
