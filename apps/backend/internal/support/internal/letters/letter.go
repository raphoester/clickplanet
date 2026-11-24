package letters

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpmail"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/conversations"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes"
)

var (
	ErrInvalidLetter  = errors.New("the letter is invalid")
	ErrInvalidSubject = errors.New("a letter's subject is 1 to 200 characters, none of them a control character")
	ErrUnknownLetter  = errors.New("no letter kept under this id")
	ErrNotALetter     = errors.New("the address names no letter")
)

const maxSubjectLength = 200

// A letter's address under the inbox is its id after this tag, so a reply to it is never read as a note's.
const tag = "l-"

type LetterID uuid.UUID

func LetterIDOf(value string) (LetterID, error) {
	id, err := uuid.Parse(value)
	if err != nil || id == uuid.Nil {
		return LetterID{}, fmt.Errorf("%w: %q", ErrUnknownLetter, value)
	}
	return LetterID(id), nil
}

func (id LetterID) String() string {
	return uuid.UUID(id).String()
}

type IDProvider interface {
	NewID() (LetterID, error)
}

// AddressOf is the letter's own address under the inbox: a reply to it comes back to the letter.
func AddressOf(inbox notes.Inbox, id LetterID) cpmail.Address {
	return inbox.Tagged(tag + id.String())
}

// LetterOf is the letter an address under the inbox names, or ErrNotALetter.
func LetterOf(inbox notes.Inbox, recipient string) (LetterID, error) {
	found, err := inbox.TagOf(recipient)
	if err != nil {
		return LetterID{}, fmt.Errorf("%w: %w", ErrNotALetter, err)
	}
	named, ok := strings.CutPrefix(found, tag)
	if !ok {
		return LetterID{}, fmt.Errorf("%w: %q", ErrNotALetter, recipient)
	}
	id, err := LetterIDOf(named)
	if err != nil {
		return LetterID{}, fmt.Errorf("%w: %w", ErrNotALetter, err)
	}
	return id, nil
}

// A mail of the operator's in the letter, as it was mailed.
type Sent struct {
	Text string
	Mail cpmail.MessageID
	At   time.Time
}

// Mail is the next mail of a letter, as it goes out.
type Mail struct {
	Letter  LetterID
	To      cpmail.Address
	Subject string
	Text    string
	// The earlier mails of the letter, oldest first: the player's client keeps them in one thread.
	Thread []cpmail.MessageID
}

// Letter is what the operator writes to a player first, the operator's further mails and the player's replies.
type Letter struct {
	id      LetterID
	account notes.AccountID
	to      cpmail.Address
	subject string
	// Never empty: a letter is kept once its first mail went.
	sent    []Sent
	replies []conversations.Reply
	closed  time.Time
}

// Draft is a letter about to be written, to an account at its address, under a subject: kept once its first mail goes.
type Draft struct {
	id      LetterID
	account notes.AccountID
	to      cpmail.Address
	subject string
}

func NewDraft(id LetterID, account notes.AccountID, to cpmail.Address, subject string) (Draft, error) {
	cleaned, err := subjectOf(subject)
	if err != nil {
		return Draft{}, err
	}
	if to == "" {
		return Draft{}, notes.ErrNoReplyAddress
	}
	return Draft{id: id, account: account, to: to, subject: cleaned}, nil
}

func (d Draft) Mail(text string) (Mail, error) {
	return Letter{id: d.id, account: d.account, to: d.to, subject: d.subject}.Mail(text)
}

// Sent is the letter its first mail made.
func (d Draft) Sent(text string, mail cpmail.MessageID, at time.Time) (Letter, error) {
	return Snapshot{
		ID: d.id, Account: d.account, To: d.to, Subject: d.subject, Sent: []Sent{{Text: text, Mail: mail, At: at}},
	}.Restore()
}

// A letter anonymized has no address: nothing goes out, and no reply comes back to it.
func (l Letter) AddressError() error {
	if l.to == "" {
		return notes.ErrNoReplyAddress
	}
	return nil
}

// Mail is the letter's next mail: under its subject first, then in reply to it.
func (l Letter) Mail(text string) (Mail, error) {
	if err := l.AddressError(); err != nil {
		return Mail{}, err
	}
	cleaned, err := notes.AnswerTextOf(text)
	if err != nil {
		return Mail{}, err //nolint:wrapcheck // the handler reads the rule's own error.
	}
	subject := l.subject
	if len(l.sent) > 0 {
		subject = "Re: " + subject
	}
	return Mail{Letter: l.id, To: l.to, Subject: subject, Text: cleaned, Thread: l.Thread()}, nil
}

func (l Letter) Sent(text string, mail cpmail.MessageID, at time.Time) (Letter, error) {
	sent := l.TakeSnapshot()
	sent.Sent = append(sent.Sent, Sent{Text: text, Mail: mail, At: at})
	return sent.Restore()
}

func (l Letter) Replied(reply conversations.Reply) (Letter, error) {
	if slices.ContainsFunc(l.replies, func(kept conversations.Reply) bool { return kept.Mail == reply.Mail }) {
		return Letter{}, fmt.Errorf("%w: %s", conversations.ErrReceivedAlready, reply.Mail.Header())
	}
	replied := l.TakeSnapshot()
	replied.Replies = append(replied.Replies, reply)
	return replied.Restore()
}

// Closed with no further mail: the letter waits for nothing until the player writes again.
func (l Letter) Closed(at time.Time) Letter {
	l.closed = at
	return l
}

// Anonymized no longer names the player: no account, no address, no reply's sender. What was said stays.
func (l Letter) Anonymized() Letter {
	l.account, l.to = notes.AccountID{}, ""
	replies := slices.Clone(l.replies)
	for n := range replies {
		replies[n].From = ""
	}
	l.replies = replies
	return l
}

// Every mail of the letter, oldest first.
func (l Letter) Thread() []cpmail.MessageID {
	type mail struct {
		id cpmail.MessageID
		at time.Time
	}
	mails := make([]mail, 0, len(l.sent)+len(l.replies))
	for _, sent := range l.sent {
		mails = append(mails, mail{id: sent.Mail, at: sent.At})
	}
	for _, reply := range l.replies {
		mails = append(mails, mail{id: reply.Mail, at: reply.At})
	}
	slices.SortStableFunc(mails, func(a, b mail) int { return a.at.Compare(b.at) })
	thread := make([]cpmail.MessageID, 0, len(mails))
	for _, m := range mails {
		thread = append(thread, m.id)
	}
	return thread
}

func (l Letter) TakeSnapshot() Snapshot {
	return Snapshot{
		ID: l.id, Account: l.account, To: l.to, Subject: l.subject, Sent: slices.Clone(l.sent),
		Replies: slices.Clone(l.replies), ClosedAt: l.closed,
	}
}

// The state of a Letter, for a store. It holds no rule: Restore does.
type Snapshot struct {
	ID LetterID
	// Zero once anonymized.
	Account notes.AccountID
	// Empty once anonymized.
	To      cpmail.Address
	Subject string
	// Each in the order it went; the first is the letter itself.
	Sent    []Sent
	Replies []conversations.Reply
	// Zero if never closed.
	ClosedAt time.Time
}

func (s Snapshot) Restore() (Letter, error) {
	if s.ID == (LetterID{}) {
		return Letter{}, fmt.Errorf("%w: no id", ErrInvalidLetter)
	}
	if (s.Account == notes.AccountID{}) != (s.To == "") {
		return Letter{}, fmt.Errorf("%w: an address with no account, or an account with no address", ErrInvalidLetter)
	}
	if s.To != "" {
		if _, err := cpmail.AddressOf(string(s.To)); err != nil {
			return Letter{}, fmt.Errorf("%w: %w", ErrInvalidLetter, err)
		}
	}
	subject, err := subjectOf(s.Subject)
	if err != nil {
		return Letter{}, err
	}
	if len(s.Sent) == 0 {
		return Letter{}, fmt.Errorf("%w: no mail went", ErrInvalidLetter)
	}

	sent := make([]Sent, 0, len(s.Sent))
	for _, mail := range s.Sent {
		text, err := notes.AnswerTextOf(mail.Text)
		if err != nil {
			return Letter{}, err //nolint:wrapcheck // the handler reads the rule's own error.
		}
		if err := mailError(mail.Mail, mail.At); err != nil {
			return Letter{}, err
		}
		sent = append(sent, Sent{Text: text, Mail: mail.Mail, At: mail.At})
	}
	var replies []conversations.Reply
	for _, reply := range s.Replies {
		if err := mailError(reply.Mail, reply.At); err != nil {
			return Letter{}, err
		}
		replies = append(replies, conversations.Reply{
			Mail: reply.Mail, From: reply.From, Text: conversations.ReplyTextOf(reply.Text), At: reply.At,
		})
	}
	return Letter{
		id: s.ID, account: s.Account, to: s.To, subject: subject, sent: sent, replies: replies, closed: s.ClosedAt,
	}, nil
}

func subjectOf(value string) (string, error) {
	subject := strings.TrimSpace(value)
	if !utf8.ValidString(subject) || subject == "" || utf8.RuneCountInString(subject) > maxSubjectLength ||
		strings.ContainsFunc(subject, unicode.IsControl) {
		return "", fmt.Errorf("%w: %q", ErrInvalidSubject, value)
	}
	return subject, nil
}

func mailError(id cpmail.MessageID, at time.Time) error {
	if _, err := cpmail.MessageIDOf(string(id)); err != nil {
		return fmt.Errorf("%w: %w", ErrInvalidLetter, err)
	}
	if at.IsZero() {
		return fmt.Errorf("%w: a mail with no time", ErrInvalidLetter)
	}
	return nil
}

type Storage interface {
	// ErrUnknownLetter for one never kept.
	Letter(ctx context.Context, id LetterID) (Letter, error)
	// Every letter to the account.
	To(ctx context.Context, account notes.AccountID) ([]Letter, error)
	// The letters first written before the time that still name their player.
	WrittenBefore(ctx context.Context, at time.Time) ([]Letter, error)
	Save(ctx context.Context, letters ...Letter) error
}
