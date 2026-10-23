package answer_quiz_usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses/usecases/answer_quiz_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
)

func asPlayer(t *testing.T) context.Context {
	t.Helper()

	return cpctx.AddAccountToContext(cpctx.AddIPToContext(t.Context(), "1.2.3.4"), string(guest))
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

const guest bonuses.Holder = "a-guest"

type fixture struct {
	charges bonuses.Charges
	store   *bonuses.MemoryStore
}

func newUseCase(answered bonuses.Answered, known bool) (*answer_quiz_usecase.UseCase, *fakeRegistry, fixture) {
	registry := &fakeRegistry{answered: answered, known: known}
	store := bonuses.NewMemoryStore()
	charges := bonuses.NewCharges(store, bonuses.ChargesConfig{SpreadClicks: 8, Enclosures: 3})

	return answer_quiz_usecase.New(registry, charges), registry, fixture{charges: charges, store: store}
}

func TestARightAnswerGrantsTheChargeAndTellsThePlanet(t *testing.T) {
	useCase, registry, f := newUseCase(bonuses.Answered{
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
	assert.Equal(t, 3, f.store.Kept(guest).SpreadClicks, "the charge is held by the account that answered")

	require.Len(t, registry.published, 1)
	assert.Equal(t, bonuses.Taken{
		CountryID: "bg", Kind: bonuses.KindSpreadClicks, QuizSubject: "ee",
	}, registry.published[0], "the planet hears who won it, and what the question was about")

	assert.Equal(t, bonuses.Entrant("1.2.3.4"), registry.forEntrant, "a guest's answer is settled against its scope")
}

func TestAWrongAnswerChangesNothingAndTellsNobody(t *testing.T) {
	useCase, registry, f := newUseCase(bonuses.Answered{CorrectChoice: 1}, true)

	out, err := useCase.Execute(asPlayer(t), answer_quiz_usecase.In{Token: "t", Choice: 0, CountryID: "bg"})
	require.NoError(t, err, "a wrong answer landed; it is not a failed call")

	assert.False(t, out.Correct)
	assert.Equal(t, 1, out.CorrectChoice, "it still says which one was right")
	assert.Empty(t, out.Kind)
	assert.Zero(t, out.Amount)
	assert.True(t, out.Held.Empty())
	assert.True(t, f.store.Kept(guest).Empty())
	assert.Empty(t, registry.published, "nothing happened, so the planet is told nothing")
}

func TestAQuizThatIsNotThisCallersIsNotFound(t *testing.T) {
	useCase, registry, _ := newUseCase(bonuses.Answered{}, false)

	_, err := useCase.Execute(asPlayer(t), answer_quiz_usecase.In{Token: "t"})
	require.ErrorIs(t, err, answer_quiz_usecase.ErrNoSuchQuiz)
	assert.Empty(t, registry.published)
}

func TestTheAmountIsWhatWasKeptNotWhatWasDrawn(t *testing.T) {
	useCase, _, f := newUseCase(bonuses.Answered{
		Correct: true,
		Reward:  bonuses.Reward{Kind: bonuses.KindEncloseClicks, Amount: 3},
	}, true)
	require.NoError(t, f.charges.Grant(t.Context(), guest, bonuses.KindEncloseClicks, 2))

	out, err := useCase.Execute(asPlayer(t), answer_quiz_usecase.In{Token: "t"})
	require.NoError(t, err)

	assert.Equal(t, 1, out.Amount, "the stack holds 3, and 2 were already in it")
	assert.Equal(t, 3, out.Held.Enclosures)
}

func TestAStoreThatFailsFailsTheAnswerAndTellsNobody(t *testing.T) {
	useCase, registry, f := newUseCase(bonuses.Answered{
		Correct: true,
		Reward:  bonuses.Reward{Kind: bonuses.KindBomb, Amount: 1},
	}, true)
	down := errors.New("postgres is down")
	f.store.FailWith(down)

	_, err := useCase.Execute(asPlayer(t), answer_quiz_usecase.In{Token: "t", CountryID: "bg"})

	require.ErrorIs(t, err, down)
	assert.Empty(t, registry.published)
}
