package quizzes_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/quizzes"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
)

func loaded(t *testing.T, config quizzes.Config) *quizzes.Bank {
	t.Helper()

	bank, err := quizzes.Load(config, nil)
	require.NoError(t, err)

	return bank
}

func TestTheShippedBankLoads(t *testing.T) {
	bank := loaded(t, quizzes.Config{})

	assert.Positive(t, bank.Size())
	assert.Greater(t, bank.Subjects(), 100, "a bank that asks about a handful of countries is a stale bank")
	assert.Regexp(t, `^bank-[0-9a-f]{8}\.json$`, bank.Name())
}

func TestADrawIsThreeChoicesWithOneOfThemRight(t *testing.T) {
	bank := loaded(t, quizzes.Config{})

	for range 200 {
		round := bank.Draw()

		require.Len(t, round.Options, quizzes.Choices)
		require.InDelta(t, 0, round.Correct, float64(quizzes.Choices-1))
		require.Equal(t, round.Question.Answer, round.Options[round.Correct])
		require.True(t, round.Correctly(round.Correct))
		require.NotEmpty(t, round.Question.Text)

		require.Equal(t, quizzes.Choices, cpcolls.NewSet(round.Options...).Len())
	}
}

func TestTheSameQuestionIsNotAlwaysTheSameThreeChoices(t *testing.T) {
	bank := loaded(t, quizzes.Config{})

	first := bank.Draw()
	for range 5000 {
		again := bank.Draw()
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
	require.Error(t, quizzes.Question{ID: "x", Text: "?", Wrong: []string{"a", "b"}}.Validate())
	require.Error(t, quizzes.Question{ID: "x", Answer: "a", Wrong: []string{"b", "c"}}.Validate())
	require.Error(t, quizzes.Question{Text: "?", Answer: "a", Wrong: []string{"b", "c"}}.Validate())
}

func TestAQuestionWithTooFewWrongAnswersRefusesTheBoot(t *testing.T) {
	require.Error(t, quizzes.Question{ID: "x", Text: "?", Answer: "a", Wrong: []string{"b"}}.Validate())
}

func TestAnAnswerThatIsAlsoOneOfTheWrongOnesRefusesTheBoot(t *testing.T) {
	require.Error(t, quizzes.Question{
		ID: "x", Text: "?", Answer: "a", Wrong: []string{"a", "b"},
	}.Validate())
}

func TestTheShippedBankHasNoQuestionThatCannotBeAsked(t *testing.T) {
	assert.NotNil(t, loaded(t, quizzes.Config{}))
}
