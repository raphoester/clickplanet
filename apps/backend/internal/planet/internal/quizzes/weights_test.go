package quizzes_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/quizzes"
)

// board is what each country holds, as a fraction of the whole map.
type board map[string]float64

func (b board) Share(country string) float64 { return b[country] }

// The lean is a statistical claim, so these tests draw enough to see it and assert on the shape
// rather than on a number. The alternative — asserting the weight arithmetic directly — would pass
// while the draw ignored it.
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
	// One country holding a tenth of the planet, on a board where an even share is well under 1%.
	counts := subjectCounts(t, quizzes.Config{LeaderBias: 1}, board{"bg": 0.1})

	flat := subjectCounts(t, quizzes.Config{LeaderBias: 0}, board{"bg": 0.1})

	assert.Greater(t, counts["bg"], flat["bg"]*3,
		"a country holding a tenth of the map should come up far more than it does on a flat draw")
}

func TestNoCountryIsEverDrawnOutOfTheGame(t *testing.T) {
	counts := subjectCounts(t, quizzes.Config{LeaderBias: 1}, board{"bg": 0.9})

	// The floor of 1 is what keeps every country in the draw however badly it is doing, and the cap
	// on the lean is what keeps a nearly-won planet from being the only thing asked about.
	assert.Positive(t, counts["fr"], "a country holding nothing is still asked about")
	assert.Less(t, counts["bg"], draws/2,
		"even a country that has almost swept the planet does not take over the quiz")
}

func TestNoBoardIsAFlatDraw(t *testing.T) {
	// A nil Shares is a process with no map to read; the lean has nothing to lean on.
	counts := subjectCounts(t, quizzes.Config{LeaderBias: 1}, nil)
	flat := subjectCounts(t, quizzes.Config{LeaderBias: 0}, board{})

	assert.InDelta(t, len(flat), len(counts), float64(len(flat))*0.1,
		"both should reach about as many countries")
}

func TestTheDrawReadsTheBoardAsItIsNow(t *testing.T) {
	// The board moves while the process runs, so the bank reads it at every draw rather than
	// taking a snapshot at boot: a quiz that asked about last week's leaders forever would be the
	// one thing this feature must not do.
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
