package answer_quiz_usecase_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses/usecases/answer_quiz_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
)

func asPlayer(t *testing.T) context.Context {
	t.Helper()

	return cpctx.AddAccountToContext(cpctx.AddIPToContext(t.Context(), "1.2.3.4"), "a-guest")
}

type fakeRegistry struct {
	answered bonuses.Answered
	known    bool

	choice     int
	forEntrant bonuses.Entrant
	published  []bonuses.Taken
}

func (f *fakeRegistry) AnswerQuiz(_ string, entrant bonuses.Entrant, choice int) (bonuses.Answered, bool) {
	f.choice, f.forEntrant = choice, entrant

	return f.answered, f.known
}

func (f *fakeRegistry) Publish(taken bonuses.Taken) {
	f.published = append(f.published, taken)
}

type fakeCharger struct{ held bonuses.Held }

func (f *fakeCharger) Held(bonuses.Holder) bonuses.Held { return f.held }

func (f *fakeCharger) Grant(_ bonuses.Holder, kind bonuses.Kind, amount int) {
	f.held = f.held.Granted(kind, amount, bonuses.ChargesConfig{SpreadClicks: 8, Enclosures: 3})
}

func newUseCase(answered bonuses.Answered, known bool) (*answer_quiz_usecase.UseCase, *fakeRegistry, *fakeCharger) {
	registry := &fakeRegistry{answered: answered, known: known}
	charger := &fakeCharger{}

	return answer_quiz_usecase.New(registry, charger), registry, charger
}

func TestARightAnswerGrantsTheChargeAndTellsThePlanet(t *testing.T) {
	useCase, registry, charger := newUseCase(bonuses.Answered{
		Correct:       true,
		CorrectChoice: 2,
		Reward:        bonuses.Reward{Kind: bonuses.KindSpreadClicks, Amount: 3},
		Subject:       "ee",
	}, true)

	out, err := useCase.Execute(asPlayer(t), answer_quiz_usecase.In{Token: "t", Choice: 2, CountryID: "bg"})
	require.NoError(t, err)

	assert.True(t, out.Correct)
	assert.Equal(t, bonuses.KindSpreadClicks, out.Kind)
	assert.Equal(t, 3, out.Amount)
	assert.Equal(t, 3, out.Held.SpreadClicks)
	assert.Equal(t, 3, charger.held.SpreadClicks, "the charge is held by the account that answered")

	require.Len(t, registry.published, 1)
	assert.Equal(t, bonuses.Taken{
		CountryID: "bg", Kind: bonuses.KindSpreadClicks, QuizSubject: "ee",
	}, registry.published[0], "the planet hears who won it, and what the question was about")

	assert.Equal(t, bonuses.Entrant("1.2.3.4"), registry.forEntrant, "a guest's answer is settled against its scope")
}

func TestAWrongAnswerChangesNothingAndTellsNobody(t *testing.T) {
	useCase, registry, charger := newUseCase(bonuses.Answered{CorrectChoice: 1}, true)

	out, err := useCase.Execute(asPlayer(t), answer_quiz_usecase.In{Token: "t", Choice: 0, CountryID: "bg"})
	require.NoError(t, err, "a wrong answer landed; it is not a failed call")

	assert.False(t, out.Correct)
	assert.Equal(t, 1, out.CorrectChoice, "it still says which one was right")
	assert.Empty(t, out.Kind)
	assert.Zero(t, out.Amount)
	assert.True(t, out.Held.Empty())
	assert.True(t, charger.held.Empty())
	assert.Empty(t, registry.published, "nothing happened, so the planet is told nothing")
}

func TestAQuizThatIsNotThisCallersIsNotFound(t *testing.T) {
	useCase, registry, _ := newUseCase(bonuses.Answered{}, false)

	_, err := useCase.Execute(asPlayer(t), answer_quiz_usecase.In{Token: "t"})
	require.ErrorIs(t, err, answer_quiz_usecase.ErrNoSuchQuiz)
	assert.Empty(t, registry.published)
}

func TestTheAmountIsWhatWasKeptNotWhatWasDrawn(t *testing.T) {
	useCase, _, charger := newUseCase(bonuses.Answered{
		Correct: true,
		Reward:  bonuses.Reward{Kind: bonuses.KindEncloseClicks, Amount: 3},
	}, true)
	charger.held = bonuses.Held{Enclosures: 2}

	out, err := useCase.Execute(asPlayer(t), answer_quiz_usecase.In{Token: "t"})
	require.NoError(t, err)

	assert.Equal(t, 1, out.Amount, "the stack holds 3, and 2 were already in it")
	assert.Equal(t, 3, out.Held.Enclosures)
}
