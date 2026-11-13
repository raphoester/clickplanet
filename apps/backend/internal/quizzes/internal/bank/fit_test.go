package bank_test

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/quizzes/internal/bank"
	"github.com/raphoester/clickplanet.lol-backend/internal/quizzes/internal/ratings"
)

type asked struct {
	id      string
	subject string
	prior   float64
}

// bankOf is a bank of the test's own: each question names its subject, so a draw with no lean picks a subject flat.
func bankOf(t *testing.T, rated bank.Rated, questions ...asked) *bank.Bank {
	t.Helper()

	entries := make([]map[string]any, 0, len(questions))
	for _, question := range questions {
		entry := map[string]any{
			"id": question.id, "subject": question.subject, "namesSubject": true,
			"text": "Which is " + question.id + "?", "answer": question.id, "wrong": []string{"a", "b"},
		}
		if question.prior > 0 {
			entry["prior"] = question.prior
		}
		entries = append(entries, entry)
	}
	blob, err := json.Marshal(map[string]any{"format": 1, "questions": entries})
	require.NoError(t, err)

	questionBank, err := bank.LoadFixture(blob, nil, rated)
	require.NoError(t, err)
	return questionBank
}

// measured is a question its answers moved to this chance.
func measured(chance float64, answers int) ratings.Rating {
	return ratings.RatingOf(ratings.LevelOf(chance), answers, int(chance*float64(answers)))
}

func shares(questionBank *bank.Bank, fit bank.Fit, draws int) map[string]float64 {
	counts := map[string]float64{}
	for range draws {
		counts[string(questionBank.Draw(0, fit).Question.ID)] += 1.0 / float64(draws)
	}
	return counts
}

var aimed = bank.Fit{Target: 0.7, Width: 1, Floor: 0.42, FloorAfter: 15}

func TestWithNoWidthTheDrawIsTheDrawOfBefore(t *testing.T) {
	questionBank := bankOf(t, ratings.NewBook(map[ratings.QuestionID]ratings.Rating{"hard": measured(0.35, 100)}),
		asked{id: "hard", subject: "ee"}, asked{id: "easy", subject: "ee", prior: 0.95})

	drawn := shares(questionBank, bank.Fit{Target: 0.7, Floor: 0.42, FloorAfter: 15, Explore: 0.1}, 8000)

	assert.InDelta(t, 0.5, drawn["hard"], 0.03, "off: no floor, and no question weighs more for how hard it is")
	assert.InDelta(t, 0.5, drawn["easy"], 0.03)
}

func TestTheDrawLeansToQuestionsNearTheTarget(t *testing.T) {
	questionBank := bankOf(t, ratings.NewBook(nil),
		asked{id: "niche", subject: "ee", prior: 0.4},
		asked{id: "fair", subject: "ee", prior: 0.7},
		asked{id: "obvious", subject: "ee", prior: 0.95})

	drawn := shares(questionBank, aimed, 20000)

	// Each weighs exp(−offset²/2), its rating's offset from the target's in widths: 2.40, 0 and 2.31.
	assert.InDelta(t, 0.888, drawn["fair"], 0.02)
	assert.InDelta(t, 0.050, drawn["niche"], 0.01)
	assert.InDelta(t, 0.062, drawn["obvious"], 0.01)
}

func TestAWiderBellLeansLess(t *testing.T) {
	questionBank := bankOf(t, ratings.NewBook(nil),
		asked{id: "niche", subject: "ee", prior: 0.4}, asked{id: "fair", subject: "ee", prior: 0.7})

	narrow, wide := aimed, aimed
	wide.Width = 4

	assert.Greater(t, shares(questionBank, wide, 8000)["niche"], shares(questionBank, narrow, 8000)["niche"]+0.2)
}

func TestTheRatingAnswersMovedWeighsAndNotThePrior(t *testing.T) {
	questionBank := bankOf(t, ratings.NewBook(map[ratings.QuestionID]ratings.Rating{"looked-easy": measured(0.7, 40)}),
		asked{id: "looked-easy", subject: "ee", prior: 0.95}, asked{id: "plain", subject: "ee", prior: 0.95})

	drawn := shares(questionBank, aimed, 8000)

	assert.Greater(t, drawn["looked-easy"], 0.85, "its answers say it sits on the target, whatever its prior said")
}

func TestAQuestionMeasuredUnderTheFloorIsNeverDrawn(t *testing.T) {
	questionBank := bankOf(t, ratings.NewBook(map[ratings.QuestionID]ratings.Rating{"impossible": measured(0.36, 15)}),
		asked{id: "impossible", subject: "ee"}, asked{id: "fair", subject: "ee", prior: 0.7})

	exploring := aimed
	exploring.Explore = 1

	assert.Zero(t, shares(questionBank, aimed, 4000)["impossible"])
	assert.Zero(t, shares(questionBank, exploring, 4000)["impossible"], "exploring never asks one under the floor either")
}

func TestAPriorAloneLeavesNoQuestionOut(t *testing.T) {
	questionBank := bankOf(t, ratings.NewBook(map[ratings.QuestionID]ratings.Rating{"few": measured(0.36, 14)}),
		asked{id: "feared", subject: "ee", prior: 0.35}, asked{id: "few", subject: "ee"},
		asked{id: "fair", subject: "ee", prior: 0.7})

	exploring := aimed
	exploring.Explore = 1
	drawn := shares(questionBank, exploring, 6000)

	assert.InDelta(t, 1.0/3, drawn["feared"], 0.03, "a prior under the floor only weighs it less, so a wrong one is corrected")
	assert.InDelta(t, 1.0/3, drawn["few"], 0.03, "fourteen answers are not yet a measure")
}

func TestAnExploringDrawIgnoresHowHardAQuestionIs(t *testing.T) {
	questionBank := bankOf(t, ratings.NewBook(nil),
		asked{id: "niche", subject: "ee", prior: 0.4}, asked{id: "fair", subject: "ee", prior: 0.7})

	tenth := aimed
	tenth.Explore = 0.1

	assert.InDelta(t, 0.05+0.9*0.0564/1.0564, shares(questionBank, tenth, 20000)["niche"], 0.01,
		"a tenth of the draws is flat, and the rest weighs it 0.056 to the other's 1")
}

func TestTheSubjectIsPickedAsBefore(t *testing.T) {
	questionBank := bankOf(t, ratings.NewBook(nil),
		asked{id: "ee-off", subject: "ee", prior: 0.99}, asked{id: "ee-off-too", subject: "ee", prior: 0.99},
		asked{id: "lv-on", subject: "lv", prior: 0.7})

	drawn := shares(questionBank, aimed, 8000)

	assert.InDelta(t, 0.5, drawn["ee-off"]+drawn["ee-off-too"], 0.03,
		"the weighing happens inside the subject the lean picked, never across subjects")
}

func TestAPoolWithNothingToDrawIsLeftOut(t *testing.T) {
	questionBank := bankOf(t, ratings.NewBook(map[ratings.QuestionID]ratings.Rating{"ee-impossible": measured(0.35, 30)}),
		asked{id: "ee-impossible", subject: "ee"}, asked{id: "lv-fair", subject: "lv", prior: 0.7})

	assert.InDelta(t, 1, shares(questionBank, aimed, 2000)["lv-fair"], 1e-9, "the lean draws again among the pools left")
}

func TestNothingToDrawAnywhereIsAFlatDrawOfTheBank(t *testing.T) {
	kept := map[ratings.QuestionID]ratings.Rating{}
	questions := make([]asked, 0, 4)
	for i := range 4 {
		id := fmt.Sprintf("impossible-%d", i)
		kept[ratings.QuestionID(id)] = measured(0.35, 30)
		questions = append(questions, asked{id: id, subject: []string{"ee", "lv"}[i%2]})
	}
	questionBank := bankOf(t, ratings.NewBook(kept), questions...)

	drawn := shares(questionBank, aimed, 4000)

	assert.Len(t, drawn, 4, "a quiz is still asked, from the whole bank")
}
