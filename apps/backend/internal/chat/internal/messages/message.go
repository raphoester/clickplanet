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
	room    Room
	country string
	text    string
	gif     Gif
	quote   Quote
}

type Snapshot struct {
	ID      MessageID
	SentAt  time.Time
	Account AccountID
	Room    Room
	Country string
	Text    string
	Gif     Gif
	Quote   Quote
}

type Quote struct {
	MessageID MessageID
	Account   AccountID
	Text      string
	Gif       Gif
}

func (s Snapshot) Restore() (Message, error) {
	if s.ID == "" {
		return Message{}, fmt.Errorf("%w: no id", ErrInvalidMessage)
	}
	if s.Room == "" {
		return Message{}, fmt.Errorf("%w: no room", ErrInvalidMessage)
	}

	return Message{
		id:      s.ID,
		sentAt:  s.SentAt,
		account: s.Account,
		room:    s.Room,
		country: s.Country,
		text:    s.Text,
		gif:     s.Gif,
		quote:   s.Quote,
	}, nil
}

func (m Message) TakeSnapshot() Snapshot {
	return Snapshot{
		ID:      m.id,
		SentAt:  m.sentAt,
		Account: m.account,
		Room:    m.room,
		Country: m.country,
		Text:    m.text,
		Gif:     m.gif,
		Quote:   m.quote,
	}
}

// A guest account is free, and a private tab is a new one, so a mute does not stop a guest who posts GIFs.
func (m Message) CheckSender(author Author) error {
	if author.guest && !m.gif.None() {
		return ErrGuestGif
	}
	return nil
}

func (m Message) In(room Room) bool {
	return m.room == room
}

// Only in its own room: a message of another room is not shown there.
func (m Message) Answering(answered Message) (Message, error) {
	if answered.room != m.room {
		return Message{}, fmt.Errorf("%w: %q is in another room", ErrNotShown, answered.id)
	}
	m.quote = Quote{MessageID: answered.id, Account: answered.account, Text: answered.text, Gif: answered.gif}
	return m, nil
}

var (
	ErrInvalidMessage = errors.New("invalid chat message")
	ErrNoMessage      = errors.New("the account sent no message")
	ErrNotShown       = errors.New("the message is not shown")
)

type Storage interface {
	Append(ctx context.Context, record Record) error
	// Among the newest of its own room.
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
