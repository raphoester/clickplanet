package quizzes

import (
	"fmt"
	"strings"
)

type Question struct {
	ID string `json:"id"`

	Subject string `json:"subject"`

	NamesSubject bool `json:"namesSubject"`

	Ask string `json:"ask"`

	Text   string   `json:"text"`
	Answer string   `json:"answer"`
	Wrong  []string `json:"wrong"`
}

const Choices = 3

// Correct never leaves the server: only the text and Options cross the wire.
type Round struct {
	Question Question

	Options []string

	Correct int
}

func (r Round) Correctly(choice int) bool {
	return choice == r.Correct
}

func (q Question) Validate() error {
	switch {
	case strings.TrimSpace(q.ID) == "":
		return fmt.Errorf("a question has no id")
	case strings.TrimSpace(q.Text) == "":
		return fmt.Errorf("%s: no question", q.ID)
	case strings.TrimSpace(q.Answer) == "":
		return fmt.Errorf("%s: no answer", q.ID)
	case len(q.Wrong) < Choices-1:
		return fmt.Errorf("%s: %d wrong answers, a round of %d needs %d", q.ID, len(q.Wrong), Choices, Choices-1)
	}

	for _, wrong := range q.Wrong {
		if wrong == q.Answer {
			return fmt.Errorf("%s: %q is both the answer and one of the wrong ones", q.ID, wrong)
		}
	}

	return nil
}
