package notes

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/google/uuid"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpmail"
)

type NoteID uuid.UUID

func (id NoteID) String() string {
	return uuid.UUID(id).String()
}

func NoteIDOf(value string) (NoteID, error) {
	id, err := uuid.Parse(value)
	if err != nil || id == uuid.Nil {
		return NoteID{}, fmt.Errorf("%w: %q", ErrInvalidNoteID, value)
	}
	return NoteID(id), nil
}

type Kind string

const (
	KindSuggestion Kind = "suggestion"
	KindBug        Kind = "bug"
)

func (k Kind) known() bool {
	return k == KindSuggestion || k == KindBug
}

var (
	ErrInvalidNote    = errors.New("the note is invalid")
	ErrNoReplyAddress = errors.New("the note has no address to answer at")
	ErrUnknownNote    = errors.New("no note kept under this id")
	ErrInvalidNoteID  = errors.New("not a note id")
	ErrNotSent        = errors.New("the mail could not be sent")
)

type Note struct {
	id           NoteID
	kind         Kind
	account      AccountID
	text         string
	diagnostics  Diagnostics
	attachments  []Attachment
	replyAddress cpmail.Address
	at           time.Time
}

// What a player sends with a note, whatever its kind.
type Sent struct {
	Account      AccountID
	Text         string
	Diagnostics  Diagnostics
	ReplyAddress cpmail.Address
}

func NewSuggestion(id NoteID, note Sent, at time.Time) (Note, error) {
	return sent(Snapshot{
		ID: id, Kind: KindSuggestion, Account: note.Account, Text: note.Text, Diagnostics: note.Diagnostics,
		ReplyAddress: note.ReplyAddress, At: at,
	})
}

func NewBugReport(id NoteID, note Sent, at time.Time) (Note, error) {
	return sent(Snapshot{
		ID: id, Kind: KindBug, Account: note.Account, Text: note.Text, Diagnostics: note.Diagnostics,
		ReplyAddress: note.ReplyAddress, At: at,
	})
}

// A note sent has a sender and an address to answer at. A note read back may have neither: it was anonymized.
func sent(note Snapshot) (Note, error) {
	if note.Account == (AccountID{}) {
		return Note{}, fmt.Errorf("%w: no account", ErrInvalidNote)
	}
	if note.ReplyAddress == "" {
		return Note{}, ErrNoReplyAddress
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
	n.replyAddress = ""
	return n
}

func IDsOf(sent []Note) []NoteID {
	ids := make([]NoteID, 0, len(sent))
	for _, note := range sent {
		ids = append(ids, note.id)
	}
	return ids
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
		Attachments: slices.Clone(n.attachments), ReplyAddress: n.replyAddress, At: n.at,
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
	// Empty once the note is anonymized, and for a note sent before addresses were kept.
	ReplyAddress cpmail.Address
	At           time.Time
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

	address := s.ReplyAddress
	if address != "" {
		if address, err = cpmail.AddressOf(string(address)); err != nil {
			return Note{}, fmt.Errorf("%w: the reply address: %w", ErrInvalidNote, err)
		}
	}

	return Note{
		id: s.ID, kind: s.Kind, account: s.Account, text: text, diagnostics: s.Diagnostics.cleaned(),
		attachments: attachments, replyAddress: address, at: s.At,
	}, nil
}

// A storage finds notes and saves them. It holds no rule: what changes a note is a method of Note.
type Storage interface {
	// A note saved again replaces itself. All of them, or none.
	Save(ctx context.Context, notes ...Note) error
	// ErrUnknownNote for an id no note has.
	Note(ctx context.Context, id NoteID) (Note, error)
	// Oldest first.
	SentBy(ctx context.Context, account AccountID) ([]Note, error)
	// The notes sent before the cutoff that still have their sender: oldest first, limit at most.
	WithSenderBefore(ctx context.Context, cutoff time.Time, limit int) ([]Note, error)
	// The notes that have their sender and no address to answer at: oldest first.
	Unaddressed(ctx context.Context) ([]Note, error)
}
