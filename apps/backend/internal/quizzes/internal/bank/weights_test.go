package bank_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/quizzes/internal/bank"
)

type board map[string]float64

func (b board) Share(country string) float64 { return b[country] }

const draws = 30000

func subjectCounts(t *testing.T, lean float64, shares bank.Shares) map[string]int {
	t.Helper()

	questionBank, err := bank.Load(shares, nil)
	require.NoError(t, err)

	counts := map[string]int{}
	for range draws {
		counts[questionBank.Draw(lean, bank.Fit{}).Question.Subject]++
	}

	return counts
}

func drawsWhere(t *testing.T, lean float64, shares bank.Shares, n int, match func(bank.Question) bool) int {
	t.Helper()

	questionBank, err := bank.Load(shares, nil)
	require.NoError(t, err)

	matched := 0
	for range n {
		if match(questionBank.Draw(lean, bank.Fit{}).Question) {
			matched++
		}
	}

	return matched
}

func TestALeadingCountryIsNamedMoreOften(t *testing.T) {
	named := func(question bank.Question) bool { return question.Subject == "bg" && question.NamesSubject }

	leaned := drawsWhere(t, 1, board{"bg": 0.1}, draws, named)
	flat := drawsWhere(t, 0, board{"bg": 0.1}, draws, named)

	assert.Greater(t, leaned, flat*3,
		"a country holding a tenth of the map should come up far more than it does on a flat draw")
}

func TestALeadingCountryIsNotTheAnswerMoreOften(t *testing.T) {
	const many = 100000
	answered := func(question bank.Question) bool { return question.Answer == "Bulgaria" }

	leaned := drawsWhere(t, 1, board{"bg": 0.1}, many, answered)
	flat := drawsWhere(t, 0, board{"bg": 0.1}, many, answered)

	require.Positive(t, flat)
	assert.Less(t, leaned, flat*2, "\"pick the leader\" must not be a way to answer")
}

func TestNoCountryIsEverDrawnOutOfTheGame(t *testing.T) {
	counts := subjectCounts(t, 1, board{"bg": 0.9})

	assert.Positive(t, counts["fr"], "a country holding nothing is still asked about")
	assert.Less(t, counts["bg"], draws/2,
		"even a country that has almost swept the planet does not take over the quiz")
}

func TestNoBoardIsAFlatDraw(t *testing.T) {
	counts := subjectCounts(t, 1, nil)
	flat := subjectCounts(t, 0, board{})

	assert.InDelta(t, len(flat), len(counts), float64(len(flat))*0.1,
		"both should reach about as many countries")
}

func TestTheDrawReadsTheBoardAsItIsNow(t *testing.T) {
	live := board{}
	questionBank, err := bank.Load(live, nil)
	require.NoError(t, err)

	before := 0
	for range draws {
		if questionBank.Draw(1, bank.Fit{}).Question.Subject == "bg" {
			before++
		}
	}

	live["bg"] = 0.2

	after := 0
	for range draws {
		if questionBank.Draw(1, bank.Fit{}).Question.Subject == "bg" {
			after++
		}
	}

	assert.Greater(t, after, before*3, "a country that took the lead is asked about more from then on")
}
