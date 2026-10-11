package messages

import (
	"context"
	"errors"
	"fmt"
	"time"
)

type MessageID string

type Message struct {
	id      MessageID
	sentAt  time.Time
	account AccountID
	author  Author
	country string
	text    string
	quote   Quote
}

type Snapshot struct {
	ID      MessageID
	SentAt  time.Time
	Account AccountID
	Author  Author
	Country string
	Text    string
	Quote   Quote
}

type Quote struct {
	MessageID MessageID
	Author    Author
	Text      string
}

func (s Snapshot) Restore() (Message, error) {
	if s.ID == "" {
		return Message{}, fmt.Errorf("%w: no id", ErrInvalidMessage)
	}

	return Message{
		id:      s.ID,
		sentAt:  s.SentAt,
		account: s.Account,
		author:  s.Author,
		country: s.Country,
		text:    s.Text,
		quote:   s.Quote,
	}, nil
}

func (m Message) TakeSnapshot() Snapshot {
	return Snapshot{
		ID:      m.id,
		SentAt:  m.sentAt,
		Account: m.account,
		Author:  m.author,
		Country: m.country,
		Text:    m.text,
		Quote:   m.quote,
	}
}

func (m Message) Named(author Author) Message {
	m.author = author
	return m
}

func (m Message) Answering(answered Message) Message {
	m.quote = Quote{MessageID: answered.id, Author: answered.author, Text: answered.text}
	return m
}

var (
	ErrInvalidMessage = errors.New("invalid chat message")
	ErrNoMessage      = errors.New("the account sent no message")
	ErrNotShown       = errors.New("the message is not shown")
)

type Storage interface {
	Append(ctx context.Context, record Record) error
	Shown(ctx context.Context, id MessageID, since time.Time, limit int) (bool, error)
	Message(ctx context.Context, id MessageID, since time.Time, limit int) (Message, error)
	LatestAddress(ctx context.Context, account AccountID) (string, error)
	DeleteBefore(ctx context.Context, cutoff time.Time) (int64, error)
}

type Window struct {
	size      int
	retention time.Duration
}

func NewWindow(size int, retention time.Duration) Window {
	return Window{size: size, retention: retention}
}

func (w Window) Size() int { return w.size }

func (w Window) Since(now time.Time) time.Time {
	return now.Add(-w.retention)
}
