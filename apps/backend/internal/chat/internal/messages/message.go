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
}

var ErrInvalidMessage = errors.New("invalid chat message")
