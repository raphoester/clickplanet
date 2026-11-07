package judgements

import (
	"slices"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
)

// One message as the judge reads it: who said it, where, and what, removed or not.
type Line struct {
	ID      messages.MessageID
	Account messages.AccountID
	Room    messages.Room
	SentAt  time.Time
	// One of the two.
	Text string
	Gif  messages.Gif
	// Removed already: it is read for its context, and never removed again.
	Removed bool
}

// What the judge reads of a report: never the reported line alone.
type Conversation struct {
	// The message reported; nil for a report that names none.
	Reported *Line
	// Its room's messages just before and just after it, oldest first.
	Before []Line
	After  []Line
	// The reported player's own latest messages, in every room, oldest first.
	Own []Line
}

// Each message once, oldest first.
func (c Conversation) Lines() []Line {
	var lines []Line
	for _, line := range slices.Concat(c.Before, c.reported(), c.After, c.Own) {
		if !slices.ContainsFunc(lines, func(seen Line) bool { return seen.ID == line.ID }) {
			lines = append(lines, line)
		}
	}
	slices.SortStableFunc(lines, func(a, b Line) int { return a.SentAt.Compare(b.SentAt) })
	return lines
}

func (c Conversation) reported() []Line {
	if c.Reported == nil {
		return nil
	}
	return []Line{*c.Reported}
}

// Nothing the player said is kept: there is nothing to judge.
func (c Conversation) Empty() bool {
	return c.Reported == nil && len(c.Own) == 0
}

// The lines named that the account said and nobody removed yet, each once: the model may name no other.
func (c Conversation) offending(named []messages.MessageID, account messages.AccountID) []Line {
	var found []Line
	for _, line := range c.Lines() {
		if line.Account == account && !line.Removed && slices.Contains(named, line.ID) {
			found = append(found, line)
		}
	}
	return found
}
