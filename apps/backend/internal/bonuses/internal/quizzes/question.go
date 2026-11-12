package quizzes

import (
	"fmt"
	"math"
	"strings"

	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/ratings"
)

type Question struct {
	ID ratings.QuestionID `json:"id"`

	Subject string `json:"subject"`

	NamesSubject bool `json:"namesSubject"`

	Text   string   `json:"text"`
	Answer string   `json:"answer"`
	Wrong  []string `json:"wrong"`

	// Prior is the share of players the bank expects to get it right, from a third (a lucky guess) to 1. Nil is none:
	// the question starts at FallbackPrior.
	Prior *float64 `json:"prior,omitempty"`
}

// FallbackPrior is the share a question the bank holds no prior for is expected to be answered right at: more often
// right than wrong, as most of the bank's questions are made to be.
const FallbackPrior = 0.6

// startingChance is the chance its rating starts from: its prior, else FallbackPrior.
func (q Question) startingChance() float64 {
	if q.Prior == nil {
		return FallbackPrior
	}
	return *q.Prior
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
	case strings.TrimSpace(string(q.ID)) == "":
		return fmt.Errorf("a question has no id")
	case strings.TrimSpace(q.Text) == "":
		return fmt.Errorf("%s: no question", q.ID)
	case strings.TrimSpace(q.Answer) == "":
		return fmt.Errorf("%s: no answer", q.ID)
	case len(q.Wrong) < Choices-1:
		return fmt.Errorf("%s: %d wrong answers, a round of %d needs %d", q.ID, len(q.Wrong), Choices, Choices-1)
	}

	if q.Prior != nil && (math.IsNaN(*q.Prior) || *q.Prior < 0 || *q.Prior > 1) {
		return fmt.Errorf("%s: a prior of %v is not a share between 0 and 1", q.ID, *q.Prior)
	}

	for _, wrong := range q.Wrong {
		if wrong == q.Answer {
			return fmt.Errorf("%s: %q is both the answer and one of the wrong ones", q.ID, wrong)
		}
	}

	return nil
}
