package domain

import (
	"errors"
	"time"
)

type ChatMessage struct {
	ID         string
	SentAt     time.Time
	AuthorName string
	AuthorTag  string
	CountryID  string
	Text       string
}

type ChatRecord struct {
	Message   ChatMessage
	AuthorID  string
	IP        string
	UserAgent string
}

var ErrInvalidMessage = errors.New("invalid chat message")
