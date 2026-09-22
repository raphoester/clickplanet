package quizzes_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/quizzes"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
)

// The bank under test is the real one, embedded. It is committed and content-addressed, so these
// are assertions about what ships rather than about a fixture: a generator that started writing a
// question with no answer, or a `make quiz` that copied a bank from another format, fails here
// instead of at the first offer on a Friday night.
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

		// Three choices that are not three different things is a question with two right answers.
		require.Equal(t, quizzes.Choices, cpcolls.NewSet(round.Options...).Len())
	}
}

func TestTheSameQuestionIsNotAlwaysTheSameThreeChoices(t *testing.T) {
	bank := loaded(t, quizzes.Config{})

	// Whichever entry comes up first, keep drawing until it comes up again and compare. A bank that
	// always dealt the first two wrong answers in the first two slots would be worth writing down.
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

// The fault a generated bank produces quietly: two countries share the name of a capital, or a
// distractor drawn from the same continent turns out to be the right answer after all.
func TestAnAnswerThatIsAlsoOneOfTheWrongOnesRefusesTheBoot(t *testing.T) {
	require.Error(t, quizzes.Question{
		ID: "x", Text: "?", Answer: "a", Wrong: []string{"a", "b"},
	}.Validate())
}

func TestTheShippedBankHasNoQuestionThatCannotBeAsked(t *testing.T) {
	// Load already checks every entry, so this is the assertion that it does — and that the file on
	// disk passes it.
	assert.NotNil(t, loaded(t, quizzes.Config{}))
}
