package bank_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/quizzes/internal/bank"
	"github.com/raphoester/clickplanet.lol-backend/internal/quizzes/internal/ratings"
)

// A bank of the test's own: Estonia's capital with a prior, Latvia's with none, and a field this build does not know.
const fixture = `{
  "format": 1,
  "questions": [
    {"id":"capital:ee","subject":"ee","text":"What is the capital of Estonia?","answer":"Tallinn",
     "wrong":["Riga","Vilnius"],"namesSubject":true,"prior":0.45},
    {"id":"capital:lv","subject":"lv","text":"What is the capital of Latvia?","answer":"Riga",
     "wrong":["Tallinn","Vilnius"],"namesSubject":true,"source":"a field a later generator adds"}
  ]
}`

func fixtureBank(t *testing.T, rated bank.Rated) *bank.Bank {
	t.Helper()

	questionBank, err := bank.LoadFixture([]byte(fixture), nil, rated)
	require.NoError(t, err)
	return questionBank
}

func TestAQuestionStartsAtItsPrior(t *testing.T) {
	questionBank := fixtureBank(t, ratings.NewBook(nil))

	assert.Equal(t, ratings.PriorOf(0.45), questionBank.Rating("capital:ee"))
	assert.InDelta(t, 0.45, questionBank.Rating("capital:ee").Chance(), 1e-12)
}

func TestAQuestionWithNoPriorStartsAtTheFallback(t *testing.T) {
	questionBank := fixtureBank(t, ratings.NewBook(nil))

	assert.Equal(t, ratings.PriorOf(bank.FallbackPrior), questionBank.Rating("capital:lv"))
	assert.Equal(t, ratings.PriorOf(bank.FallbackPrior), questionBank.Rating("capital:none"), "nor does one the bank lacks")
}

func TestTheRatingAnswersMovedIsReadOverThePrior(t *testing.T) {
	moved := ratings.RatingOf(2, 30, 12)
	questionBank := fixtureBank(t, ratings.NewBook(map[ratings.QuestionID]ratings.Rating{"capital:ee": moved}))

	assert.Equal(t, moved, questionBank.Rating("capital:ee"))
	assert.Equal(t, ratings.PriorOf(bank.FallbackPrior), questionBank.Rating("capital:lv"))
}

func TestAPriorThatIsNoShareRefusesTheBoot(t *testing.T) {
	for _, prior := range []float64{-0.1, 1.2} {
		require.Error(t, bank.Question{
			ID: "x", Text: "?", Answer: "a", Wrong: []string{"b", "c"}, Prior: &prior,
		}.Validate(), "%v", prior)
	}

	for _, prior := range []float64{0, 0.3, 1.0 / 3, 1} {
		require.NoError(t, bank.Question{
			ID: "x", Text: "?", Answer: "a", Wrong: []string{"b", "c"}, Prior: &prior,
		}.Validate(), "%v is a share: under a third it reads as a third, the hardest a rating goes", prior)
	}
}
