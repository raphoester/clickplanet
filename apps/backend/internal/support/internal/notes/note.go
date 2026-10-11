package notes

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type NoteID uuid.UUID

type Kind string

const KindSuggestion Kind = "suggestion"

func (k Kind) known() bool {
	return k == KindSuggestion
}

var ErrInvalidNote = errors.New("the note is invalid")

type Note struct {
	id        NoteID
	kind      Kind
	account   AccountID
	text      string
	at        time.Time
	forgotten bool
}

func NewSuggestion(id NoteID, account AccountID, text string, at time.Time) (Note, error) {
	return Snapshot{ID: id, Kind: KindSuggestion, Account: account, Text: text, At: at}.Restore()
}

// A note its account's deletion or its age took away: a storage keeps it no more.
func (n Note) Forgotten() Note {
	n.forgotten = true
	return n
}

func AllForgotten(sent []Note) []Note {
	forgotten := make([]Note, 0, len(sent))
	for _, note := range sent {
		forgotten = append(forgotten, note.Forgotten())
	}
	return forgotten
}

func (n Note) TakeSnapshot() Snapshot {
	return Snapshot{ID: n.id, Kind: n.kind, Account: n.account, Text: n.text, At: n.at, Forgotten: n.forgotten}
}

// The state of a Note, for a store. It holds no rule: Restore does.
type Snapshot struct {
	ID        NoteID
	Kind      Kind
	Account   AccountID
	Text      string
	At        time.Time
	Forgotten bool
}

func (s Snapshot) Restore() (Note, error) {
	switch {
	case s.ID == NoteID{}:
		return Note{}, fmt.Errorf("%w: no id", ErrInvalidNote)
	case !s.Kind.known():
		return Note{}, fmt.Errorf("%w: unknown kind %q", ErrInvalidNote, string(s.Kind))
	case s.Account == AccountID{}:
		return Note{}, fmt.Errorf("%w: no account", ErrInvalidNote)
	case s.At.IsZero():
		return Note{}, fmt.Errorf("%w: no time", ErrInvalidNote)
	}

	text, err := TextOf(s.Text)
	if err != nil {
		return Note{}, err
	}

	return Note{id: s.ID, kind: s.Kind, account: s.Account, text: text, at: s.At, forgotten: s.Forgotten}, nil
}

// A storage finds notes and saves them. It holds no rule: what changes a note is a method of Note.
type Storage interface {
	// A note saved again replaces itself, and a forgotten note is kept no more. All of them, or none.
	Save(ctx context.Context, notes ...Note) error
	// Oldest first.
	SentBy(ctx context.Context, account AccountID) ([]Note, error)
	// Oldest first, limit at most.
	SentBefore(ctx context.Context, cutoff time.Time, limit int) ([]Note, error)
}
