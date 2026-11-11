package answer_quiz_usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses/usecases/answer_quiz_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpredis"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var now = time.Date(2026, 10, 25, 12, 0, 0, 0, time.UTC)

var guest = bonuses.Caller{Scope: "1.2.3.4", Account: "a-guest"}

func answerOf(token string, choice int, country string) answer_quiz_usecase.In {
	return answer_quiz_usecase.In{Caller: guest, Token: token, Choice: choice, CountryID: country}
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

func newUseCase(answered bonuses.Answered, known bool) (*answer_quiz_usecase.UseCase, *fakeRegistry, *bonuses.FakeWallet) {
	registry := &fakeRegistry{answered: answered, known: known}
	wallet := bonuses.NewFakeWallet()

	return answer_quiz_usecase.New(registry, wallet, cpredis.NewRecordedEvents(), cptime.NewFixedClock(now)), registry, wallet
}

func rightFor(kind bonuses.Kind, amount int) bonuses.Answered {
	return bonuses.Answered{Correct: true, Reward: bonuses.Reward{Kind: kind, Amount: amount}}
}

func TestARightAnswerGrantsTheChargeAndTellsThePlanet(t *testing.T) {
	useCase, registry, store := newUseCase(bonuses.Answered{
		Correct:       true,
		CorrectChoice: 2,
		Reward:        bonuses.Reward{Kind: bonuses.KindSpreadClicks, Amount: 3},
		Subject:       "ee",
	}, true)

	out, err := useCase.Execute(t.Context(), answerOf("t", 2, "bg"))
	require.NoError(t, err)

	assert.True(t, out.Correct)
	assert.Equal(t, 2, out.CorrectChoice)
	assert.Equal(t, bonuses.KindSpreadClicks, out.Kind)
	assert.Equal(t, 3, out.Amount)
	assert.Equal(t, 3, out.Hand.SpreadClicks)
	assert.Equal(t, 3, store.Kept(guest.Account).SpreadClicks, "the charge is held by the account that answered")
	assert.Equal(t, bonuses.OfferGrantOf("t"), store.Payments()[0].Grant, "the payment is named by the quiz's token")

	require.Len(t, registry.published, 1)
	assert.Equal(t, bonuses.Taken{
		CountryID: "bg", Kind: bonuses.KindSpreadClicks, QuizSubject: "ee",
	}, registry.published[0], "the planet hears who won it, and what the question was about")

	assert.Equal(t, bonuses.Entrant("1.2.3.4"), registry.forEntrant, "a guest's answer is settled against its scope")
}

func TestAWrongAnswerChangesNothingAndTellsNobody(t *testing.T) {
	useCase, registry, store := newUseCase(bonuses.Answered{CorrectChoice: 1}, true)
	store.Keep(guest.Account, bonuses.Hand{Shields: 4})

	out, err := useCase.Execute(t.Context(), answerOf("t", 0, "bg"))
	require.NoError(t, err, "a wrong answer landed; it is not a failed call")

	assert.False(t, out.Correct)
	assert.Equal(t, 1, out.CorrectChoice, "it still says which one was right")
	assert.Empty(t, out.Kind)
	assert.Zero(t, out.Amount)
	assert.Equal(t, bonuses.Hand{Shields: 4}, out.Hand, "it says what the account holds")
	assert.Equal(t, bonuses.Hand{Shields: 4}, store.Kept(guest.Account))
	assert.Empty(t, registry.published, "nothing happened, so the planet is told nothing")
}

func TestEveryAnswerIsToldToTheOtherModulesRightOrWrong(t *testing.T) {
	for _, correct := range []bool{true, false} {
		events := cpredis.NewRecordedEvents()
		registry := &fakeRegistry{answered: bonuses.Answered{Correct: correct, Reward: bonuses.Reward{Kind: bonuses.KindRefill, Amount: 1}}, known: true}
		wallet := bonuses.NewFakeWallet()

		_, err := answer_quiz_usecase.New(registry, wallet, events, cptime.NewFixedClock(now)).
			Execute(t.Context(), answerOf("t", 0, ""))
		require.NoError(t, err)

		require.Len(t, events.Published(), 1)
		assert.True(t, proto.Equal(&planetv1.QuizAnswered{
			AccountId: string(guest.Account), Correct: correct, AnsweredAt: timestamppb.New(now),
		}, events.Published()[0]), "correct: %v", correct)
	}
}

func TestAQuizThatIsNotThisCallersIsNotFound(t *testing.T) {
	useCase, registry, _ := newUseCase(bonuses.Answered{}, false)

	_, err := useCase.Execute(t.Context(), answerOf("t", 0, ""))
	require.ErrorIs(t, err, answer_quiz_usecase.ErrNoSuchQuiz)
	assert.Empty(t, registry.published)
}

func TestTheAmountIsWhatWasKeptNotWhatWasDrawn(t *testing.T) {
	useCase, _, store := newUseCase(rightFor(bonuses.KindEncloseClicks, 3), true)
	store.Size(bonuses.KindEncloseClicks, 3)
	store.Keep(guest.Account, bonuses.Hand{Enclosures: 2})

	out, err := useCase.Execute(t.Context(), answerOf("t", 0, ""))
	require.NoError(t, err)

	assert.Equal(t, 1, out.Amount, "the stack holds 3, and 2 were already in it")
	assert.Equal(t, 3, out.Hand.Enclosures)
}

func TestARightAnswerIsGrantedWhenTheCallerLeavesAfterIt(t *testing.T) {
	useCase, _, store := newUseCase(rightFor(bonuses.KindBomb, 1), true)
	ctx, leave := context.WithCancel(t.Context())
	leave()

	_, err := useCase.Execute(ctx, answerOf("t", 0, ""))
	require.NoError(t, err)

	assert.Equal(t, []error{nil}, store.Gone(), "the payment runs on a context the caller's leaving does not end")
	assert.Equal(t, bonuses.Hand{Bomb: true}, store.Kept(guest.Account))
}

func TestAFailedGrantIsNotARefusalAndIsToldToNobody(t *testing.T) {
	registry := &fakeRegistry{answered: rightFor(bonuses.KindBomb, 1), known: true}
	wallet := bonuses.NewFakeWallet()
	wallet.FailWith(errors.New("postgres is down"))
	events := cpredis.NewRecordedEvents()

	_, err := answer_quiz_usecase.New(registry, wallet, events, cptime.NewFixedClock(now)).
		Execute(t.Context(), answerOf("t", 0, ""))

	require.Error(t, err)
	assert.NotErrorIs(t, err, answer_quiz_usecase.ErrNoSuchQuiz)
	assert.Empty(t, registry.published)
	assert.Empty(t, events.Published())
}
