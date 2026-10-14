package notes

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/google/uuid"
)

type NoteID uuid.UUID

type Kind string

const (
	KindSuggestion Kind = "suggestion"
	KindBug        Kind = "bug"
)

func (k Kind) known() bool {
	return k == KindSuggestion || k == KindBug
}

var ErrInvalidNote = errors.New("the note is invalid")

type Note struct {
	id          NoteID
	kind        Kind
	account     AccountID
	text        string
	diagnostics Diagnostics
	attachments []Attachment
	at          time.Time
}

func NewSuggestion(id NoteID, account AccountID, text string, diagnostics Diagnostics, at time.Time) (Note, error) {
	return sent(Snapshot{ID: id, Kind: KindSuggestion, Account: account, Text: text, Diagnostics: diagnostics, At: at})
}

func NewBugReport(id NoteID, account AccountID, text string, diagnostics Diagnostics, at time.Time) (Note, error) {
	return sent(Snapshot{ID: id, Kind: KindBug, Account: account, Text: text, Diagnostics: diagnostics, At: at})
}

// A note sent has a sender. A note read back may have none: it was anonymized.
func sent(note Snapshot) (Note, error) {
	if note.Account == (AccountID{}) {
		return Note{}, fmt.Errorf("%w: no account", ErrInvalidNote)
	}
	return note.Restore()
}

func (n Note) Attaching(files []Attachment, rules AttachmentRules) (Note, error) {
	if err := rules.refusal(files); err != nil {
		return Note{}, err
	}

	attached := n.TakeSnapshot()
	attached.Attachments = files
	return attached.Restore()
}

// The note with nothing that says who sent it: the text and the files stay.
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
	return Snapshot{
		ID: n.id, Kind: n.kind, Account: n.account, Text: n.text, Diagnostics: n.diagnostics,
		Attachments: slices.Clone(n.attachments), At: n.at,
	}
}

// The state of a Note, for a store. It holds no rule: Restore does.
type Snapshot struct {
	ID   NoteID
	Kind Kind
	// Zero once the note is anonymized.
	Account     AccountID
	Text        string
	Diagnostics Diagnostics
	// In the order they were sent.
	Attachments []Attachment
	At          time.Time
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

	attachments, err := checked(s.Attachments)
	if err != nil {
		return Note{}, err
	}

	return Note{
		id: s.ID, kind: s.Kind, account: s.Account, text: text, diagnostics: s.Diagnostics.cleaned(),
		attachments: attachments, at: s.At,
	}, nil
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
