// Package messages is the chat's one concept: a message, and the rules it passes.
package messages

import (
	"errors"
	"time"
)

type Message struct {
	ID         string
	SentAt     time.Time
	AuthorName string
	AuthorTag  string
	CountryID  string
	Text       string

	// Set on the way out, never on the way in.
	Redacted bool
}

// Redact is the message a reader sees once its author is banned: a view and not
// an edit, so the stored text stays and lifting a ban restores nothing.
func (m Message) Redact() Message {
	m.Text = ""
	m.Redacted = true
	return m
}

var ErrInvalidMessage = errors.New("invalid chat message")
