// Package quizzes is the question bank and the rounds drawn from it: what a player is asked, and
// which of the three choices in front of them is right.
//
// It knows nothing about schedules, charges or streams. Who is asked and when is the bonus
// registry's business, and what a right answer is worth is the charges'. This package answers one
// question: given a planet where some countries are winning, what should the next question be, and
// was that the right answer to it.
package quizzes

import (
	"fmt"
	"strings"
)

// Question is one entry of the bank: a thing that is asked, the answer, and more wrong answers than
// a round shows.
type Question struct {
	ID string `json:"id"`

	// The country it is about, for the banner's flag and for the leaderboard weighting. Empty for a
	// question about nowhere in particular.
	Subject string `json:"subject"`

	// Which template wrote it, for the metrics. Not shown to anyone.
	Ask string `json:"ask"`

	Text   string   `json:"text"`
	Answer string   `json:"answer"`
	Wrong  []string `json:"wrong"`
}

// Choices is how many a round puts in front of the player. Three is the whole game's balance: a
// guess is right one time in three, which is why a wrong answer costs nothing.
const Choices = 3

// Round is one question as it was put to one player: the three choices in the order they were sent,
// and which of them is right.
//
// **The right one never leaves the server.** A Round is held by the registry until it is answered;
// what crosses the wire is the text and the choices, and the index is only ever compared here.
type Round struct {
	Question Question

	// Exactly Choices, shuffled. One of them is Question.Answer.
	Options []string

	// Which of Options is Question.Answer.
	Correct int
}

// Correctly says whether choice is the right one. Out of range is wrong, not an error: a client
// that sends a fourth choice for a round of three has guessed, and guessing is a thing the game
// already has an answer for.
func (r Round) Correctly(choice int) bool {
	return choice == r.Correct
}

// Validate refuses an entry the drawer could not make a round from, so a bad bank is a boot that
// fails rather than a question nobody can answer.
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
