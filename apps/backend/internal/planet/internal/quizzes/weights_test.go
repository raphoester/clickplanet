package quizzes_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/quizzes"
)

type board map[string]float64

func (b board) Share(country string) float64 { return b[country] }

const draws = 30000

func subjectCounts(t *testing.T, config quizzes.Config, shares quizzes.Shares) map[string]int {
	t.Helper()

	bank, err := quizzes.Load(config, shares)
	require.NoError(t, err)

	counts := map[string]int{}
	for range draws {
		counts[bank.Draw().Question.Subject]++
	}

	return counts
}

func TestALeadingCountryIsAskedAboutMoreOften(t *testing.T) {
	counts := subjectCounts(t, quizzes.Config{LeaderBias: 1}, board{"bg": 0.1})

	flat := subjectCounts(t, quizzes.Config{LeaderBias: 0}, board{"bg": 0.1})

	assert.Greater(t, counts["bg"], flat["bg"]*3,
		"a country holding a tenth of the map should come up far more than it does on a flat draw")
}

func TestNoCountryIsEverDrawnOutOfTheGame(t *testing.T) {
	counts := subjectCounts(t, quizzes.Config{LeaderBias: 1}, board{"bg": 0.9})

	assert.Positive(t, counts["fr"], "a country holding nothing is still asked about")
	assert.Less(t, counts["bg"], draws/2,
		"even a country that has almost swept the planet does not take over the quiz")
}

func TestNoBoardIsAFlatDraw(t *testing.T) {
	counts := subjectCounts(t, quizzes.Config{LeaderBias: 1}, nil)
	flat := subjectCounts(t, quizzes.Config{LeaderBias: 0}, board{})

	assert.InDelta(t, len(flat), len(counts), float64(len(flat))*0.1,
		"both should reach about as many countries")
}

func TestTheDrawReadsTheBoardAsItIsNow(t *testing.T) {
	live := board{}
	bank, err := quizzes.Load(quizzes.Config{LeaderBias: 1}, live)
	require.NoError(t, err)

	before := 0
	for range draws {
		if bank.Draw().Question.Subject == "bg" {
			before++
		}
	}

	live["bg"] = 0.2

	after := 0
	for range draws {
		if bank.Draw().Question.Subject == "bg" {
			after++
		}
	}

	assert.Greater(t, after, before*3, "a country that took the lead is asked about more from then on")
}
