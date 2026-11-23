package conversations

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpmail"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes"
)

var (
	ErrInvalidConversation = errors.New("the conversation is invalid")
	ErrReceivedAlready     = errors.New("the conversation holds this mail already")
)

// The operator's answer, as it was mailed.
type Answer struct {
	Text string
	Mail cpmail.MessageID
	At   time.Time
}

// What the player wrote back by email.
type Reply struct {
	Mail cpmail.MessageID
	// The mail's From. Empty once the note is anonymized.
	From cpmail.Address
	Text string
	At   time.Time
}

// What the operator and the player wrote about one note after it was sent, and when the operator last closed it.
type Conversation struct {
	note    notes.NoteID
	answers []Answer
	replies []Reply
	closed  time.Time
}

func (c Conversation) Answered(text string, mail cpmail.MessageID, at time.Time) (Conversation, error) {
	answered := c.TakeSnapshot()
	answered.Answers = append(answered.Answers, Answer{Text: text, Mail: mail, At: at})
	return answered.Restore()
}

func (c Conversation) Replied(reply Reply) (Conversation, error) {
	if slices.ContainsFunc(c.replies, func(kept Reply) bool { return kept.Mail == reply.Mail }) {
		return Conversation{}, fmt.Errorf("%w: %s", ErrReceivedAlready, reply.Mail.Header())
	}
	replied := c.TakeSnapshot()
	replied.Replies = append(replied.Replies, reply)
	return replied.Restore()
}

// Closed with no answer: the note waits for nothing until the player writes again.
func (c Conversation) Closed(at time.Time) Conversation {
	c.closed = at
	return c
}

func (c Conversation) Anonymized() Conversation {
	anonymized := slices.Clone(c.replies)
	for n := range anonymized {
		anonymized[n].From = ""
	}
	c.replies = anonymized
	return c
}

func AllAnonymized(found []Conversation) []Conversation {
	anonymized := make([]Conversation, 0, len(found))
	for _, conversation := range found {
		anonymized = append(anonymized, conversation.Anonymized())
	}
	return anonymized
}

// Every mail about the note, oldest first.
func (c Conversation) Thread() []cpmail.MessageID {
	type mail struct {
		id cpmail.MessageID
		at time.Time
	}
	mails := make([]mail, 0, len(c.answers)+len(c.replies))
	for _, answer := range c.answers {
		mails = append(mails, mail{id: answer.Mail, at: answer.At})
	}
	for _, reply := range c.replies {
		mails = append(mails, mail{id: reply.Mail, at: reply.At})
	}
	slices.SortStableFunc(mails, func(a, b mail) int { return a.at.Compare(b.at) })

	thread := make([]cpmail.MessageID, 0, len(mails))
	for _, m := range mails {
		thread = append(thread, m.id)
	}
	return thread
}

func (c Conversation) TakeSnapshot() Snapshot {
	return Snapshot{Note: c.note, Answers: slices.Clone(c.answers), Replies: slices.Clone(c.replies), ClosedAt: c.closed}
}

// The state of a Conversation, for a store. It holds no rule: Restore does.
type Snapshot struct {
	Note notes.NoteID
	// Each in the order it came.
	Answers []Answer
	Replies []Reply
	// Zero if never closed.
	ClosedAt time.Time
}

func (s Snapshot) Restore() (Conversation, error) {
	if s.Note == (notes.NoteID{}) {
		return Conversation{}, fmt.Errorf("%w: no note", ErrInvalidConversation)
	}

	var answers []Answer
	for _, answer := range s.Answers {
		text, err := notes.AnswerTextOf(answer.Text)
		if err != nil {
			return Conversation{}, err //nolint:wrapcheck // the handler reads the rule's own error.
		}
		if err := mailError(answer.Mail, answer.At); err != nil {
			return Conversation{}, err
		}
		answers = append(answers, Answer{Text: text, Mail: answer.Mail, At: answer.At})
	}

	var replies []Reply
	for _, reply := range s.Replies {
		if err := mailError(reply.Mail, reply.At); err != nil {
			return Conversation{}, err
		}
		replies = append(replies, Reply{Mail: reply.Mail, From: reply.From, Text: ReplyTextOf(reply.Text), At: reply.At})
	}

	return Conversation{note: s.Note, answers: answers, replies: replies, closed: s.ClosedAt}, nil
}

func mailError(id cpmail.MessageID, at time.Time) error {
	if _, err := cpmail.MessageIDOf(string(id)); err != nil {
		return fmt.Errorf("%w: %w", ErrInvalidConversation, err)
	}
	if at.IsZero() {
		return fmt.Errorf("%w: a mail with no time", ErrInvalidConversation)
	}
	return nil
}

type Storage interface {
	// One per note asked, in the order asked; an empty one for a note nobody wrote about yet.
	Conversations(ctx context.Context, notes ...notes.NoteID) ([]Conversation, error)
	// All of them, or none. notes.ErrUnknownNote for a note no longer kept.
	Save(ctx context.Context, conversations ...Conversation) error
}
