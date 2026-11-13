package bank_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/quizzes/internal/bank"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
)

func loaded(t *testing.T) *bank.Bank {
	t.Helper()

	questionBank, err := bank.Load(nil, nil)
	require.NoError(t, err)

	return questionBank
}

func TestTheShippedBankLoads(t *testing.T) {
	questionBank := loaded(t)

	assert.Positive(t, questionBank.Size())
	assert.Greater(t, questionBank.Subjects(), 100, "a bank that asks about a handful of countries is a stale bank")
	assert.Regexp(t, `^[0-9a-f]{8}$`, questionBank.Version())
}

func TestADrawIsThreeChoicesWithOneOfThemRight(t *testing.T) {
	questionBank := loaded(t)

	for range 200 {
		round := questionBank.Draw(0, bank.Fit{})

		require.Len(t, round.Options, bank.Choices)
		require.InDelta(t, 0, round.Correct, float64(bank.Choices-1))
		require.Equal(t, round.Question.Answer, round.Options[round.Correct])
		require.True(t, round.Correctly(round.Correct))
		require.NotEmpty(t, round.Question.Text)

		require.Equal(t, bank.Choices, cpcolls.NewSet(round.Options...).Len())
	}
}

func TestTheSameQuestionIsNotAlwaysTheSameThreeChoices(t *testing.T) {
	questionBank := loaded(t)

	first := questionBank.Draw(0, bank.Fit{})
	for range 5000 {
		again := questionBank.Draw(0, bank.Fit{})
		if again.Question.ID != first.Question.ID {
			continue
		}
		if !assert.ObjectsAreEqual(first.Options, again.Options) {
			return
		}
	}

	t.Skip("the same entry did not come up twice in 5000 draws; the bank is bigger than this test")
}

func TestAQuestionWithNoAnswerRefusesTheBoot(t *testing.T) {
	require.Error(t, bank.Question{ID: "x", Text: "?", Wrong: []string{"a", "b"}}.Validate())
	require.Error(t, bank.Question{ID: "x", Answer: "a", Wrong: []string{"b", "c"}}.Validate())
	require.Error(t, bank.Question{Text: "?", Answer: "a", Wrong: []string{"b", "c"}}.Validate())
}

func TestAQuestionWithTooFewWrongAnswersRefusesTheBoot(t *testing.T) {
	require.Error(t, bank.Question{ID: "x", Text: "?", Answer: "a", Wrong: []string{"b"}}.Validate())
}

func TestAnAnswerThatIsAlsoOneOfTheWrongOnesRefusesTheBoot(t *testing.T) {
	require.Error(t, bank.Question{
		ID: "x", Text: "?", Answer: "a", Wrong: []string{"a", "b"},
	}.Validate())
}
