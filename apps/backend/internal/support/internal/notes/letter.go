package notes

import (
	"fmt"
	"strings"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpmail"
)

// The operator's answer to a note, as it is mailed.
type Letter struct {
	Note    NoteID
	To      cpmail.Address
	Subject string
	Text    string
	// The earlier mails about the note, oldest first: the player's client keeps them in one thread.
	Thread []cpmail.MessageID
}

var subjects = map[Kind]string{
	KindSuggestion: "Re: your idea",
	KindBug:        "Re: your bug report",
}

// The answer, then the note quoted.
func (n Note) Answer(text string, thread []cpmail.MessageID) (Letter, error) {
	if n.replyAddress == "" {
		return Letter{}, ErrNoReplyAddress
	}
	answer, err := AnswerTextOf(text)
	if err != nil {
		return Letter{}, err
	}

	quoted := "> " + strings.ReplaceAll(n.text, "\n", "\n> ")
	return Letter{
		Note: n.id, To: n.replyAddress, Subject: subjects[n.kind],
		Text:   fmt.Sprintf("%s\n\nOn %s, you wrote:\n%s\n", answer, n.at.UTC().Format("2 January 2006"), quoted),
		Thread: thread,
	}, nil
}
