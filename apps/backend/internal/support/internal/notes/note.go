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
	id      NoteID
	kind    Kind
	account AccountID
	text    string
	at      time.Time
}

func NewSuggestion(id NoteID, account AccountID, text string, at time.Time) (Note, error) {
	if account == (AccountID{}) {
		return Note{}, fmt.Errorf("%w: no account", ErrInvalidNote)
	}
	return Snapshot{ID: id, Kind: KindSuggestion, Account: account, Text: text, At: at}.Restore()
}

// The note with nothing that says who sent it: the text stays.
func (n Note) Anonymized() Note {
	n.account = AccountID{}
	return n
}

func AllAnonymized(sent []Note) []Note {
	anonymized := make([]Note, 0, len(sent))
	for _, note := range sent {
		anonymized = append(anonymized, note.Anonymized())
	}
	return anonymized
}

func (n Note) TakeSnapshot() Snapshot {
	return Snapshot{ID: n.id, Kind: n.kind, Account: n.account, Text: n.text, At: n.at}
}

// The state of a Note, for a store. It holds no rule: Restore does.
type Snapshot struct {
	ID   NoteID
	Kind Kind
	// Zero once the note is anonymized.
	Account AccountID
	Text    string
	At      time.Time
}

func (s Snapshot) Restore() (Note, error) {
	switch {
	case s.ID == NoteID{}:
		return Note{}, fmt.Errorf("%w: no id", ErrInvalidNote)
	case !s.Kind.known():
		return Note{}, fmt.Errorf("%w: unknown kind %q", ErrInvalidNote, string(s.Kind))
	case s.At.IsZero():
		return Note{}, fmt.Errorf("%w: no time", ErrInvalidNote)
	}

	text, err := TextOf(s.Text)
	if err != nil {
		return Note{}, err
	}

	return Note{id: s.ID, kind: s.Kind, account: s.Account, text: text, at: s.At}, nil
}

// A storage finds notes and saves them. It holds no rule: what changes a note is a method of Note.
type Storage interface {
	// A note saved again replaces itself. All of them, or none.
	Save(ctx context.Context, notes ...Note) error
	// Oldest first.
	SentBy(ctx context.Context, account AccountID) ([]Note, error)
	// The notes sent before the cutoff that still have their sender: oldest first, limit at most.
	WithSenderBefore(ctx context.Context, cutoff time.Time, limit int) ([]Note, error)
}
