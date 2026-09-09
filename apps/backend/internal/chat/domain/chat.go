// Package domain holds the chat bounded context. Chat shares a process, a
// transport and a country list with the tile game and nothing else: no type
// here is the tile game's, and nothing under internal/chat imports
// internal/clicks.
package domain

import (
	"errors"
	"time"
)

// ChatMessage is what every client sees: nothing here identifies a sender
// beyond the name they chose and the tag the server derived for them.
type ChatMessage struct {
	ID         string
	SentAt     time.Time
	AuthorName string
	AuthorTag  string
	CountryID  string
	Text       string
}

// ChatRecord is a ChatMessage plus the parts that only ever reach the on-disk
// log — never a client, and never the websocket fanout.
type ChatRecord struct {
	Message   ChatMessage
	AuthorID  string
	IP        string
	UserAgent string
}

// ErrInvalidMessage is what a sender is told when their message is refused. The
// reason travels wrapped for the log; the edge answers with the sentinel alone,
// so a sender learns that they were refused rather than which check tripped.
var ErrInvalidMessage = errors.New("invalid chat message")
