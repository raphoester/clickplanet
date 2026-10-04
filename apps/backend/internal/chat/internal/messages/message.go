package messages

import (
	"context"
	"errors"
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
}

func NewMessage(id MessageID, sentAt time.Time, account AccountID, country string, text string) Message {
	return Message{id: id, sentAt: sentAt, account: account, country: country, text: text}
}

func (m Message) ID() MessageID { return m.id }

func (m Message) SentAt() time.Time { return m.sentAt }

func (m Message) Account() AccountID { return m.account }

func (m Message) Author() Author { return m.author }

func (m Message) Country() string { return m.country }

func (m Message) Text() string { return m.text }

func (m Message) Named(author Author) Message {
	m.author = author
	return m
}

var ErrInvalidMessage = errors.New("invalid chat message")

type Storage interface {
	Append(ctx context.Context, record Record) error
	Shown(ctx context.Context, id MessageID, since time.Time, limit int) (bool, error)
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
