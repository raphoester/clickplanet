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
	// AuthorAdmin is whether the username it was sent under was an admin's then. Never a guest.
	AuthorAdmin bool
	CountryID   string
	Text        string
}

var ErrInvalidMessage = errors.New("invalid chat message")
