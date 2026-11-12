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

	bonusesv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/bonuses/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/bonuses/usecases/answer_quiz_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/ratings"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpredis"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var now = time.Date(2026, 10, 25, 12, 0, 0, 0, time.UTC)

var guest = bonuses.Caller{Scope: "1.2.3.4", Account: "0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11"}

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

// counted is every answer the ratings were given, with whether the caller had left by then.
type counted struct {
	answers []ratings.Answer
	gone    []error
	err     error
}

func (c *counted) Execute(ctx context.Context, answer ratings.Answer) error {
	c.answers = append(c.answers, answer)
	c.gone = append(c.gone, ctx.Err())
	return c.err
}

func newUseCase(answered bonuses.Answered, known bool) (*answer_quiz_usecase.UseCase, *fakeRegistry, *bonuses.FakeWallet) {
	useCase, registry, wallet, _ := newCountedUseCase(answered, known)
	return useCase, registry, wallet
}

func newCountedUseCase(
	answered bonuses.Answered, known bool,
) (*answer_quiz_usecase.UseCase, *fakeRegistry, *bonuses.FakeWallet, *counted) {
	registry := &fakeRegistry{answered: answered, known: known}
	wallet := bonuses.NewFakeWallet()
	answers := &counted{}

	return answer_quiz_usecase.New(registry, wallet, answers, cpredis.NewRecordedEvents(), cptime.NewFixedClock(now)),
		registry, wallet, answers
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

		_, err := answer_quiz_usecase.New(registry, wallet, &counted{}, events, cptime.NewFixedClock(now)).
			Execute(t.Context(), answerOf("t", 0, ""))
		require.NoError(t, err)

		require.Len(t, events.Published(), 1)
		assert.True(t, proto.Equal(&bonusesv1.QuizAnswered{
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

	_, err := answer_quiz_usecase.New(registry, wallet, &counted{}, events, cptime.NewFixedClock(now)).
		Execute(t.Context(), answerOf("t", 0, ""))

	require.Error(t, err)
	assert.NotErrorIs(t, err, answer_quiz_usecase.ErrNoSuchQuiz)
	assert.Empty(t, registry.published)
	assert.Empty(t, events.Published())
}

func TestEveryAnswerIsCountedForItsQuestionRightOrWrong(t *testing.T) {
	for _, correct := range []bool{true, false} {
		useCase, _, _, answers := newCountedUseCase(bonuses.Answered{
			Correct: correct, Reward: bonuses.Reward{Kind: bonuses.KindRefill, Amount: 1},
			Question: "capital:ee", After: 3 * time.Second,
		}, true)

		_, err := useCase.Execute(t.Context(), answerOf("t", 0, ""))
		require.NoError(t, err)

		account, err := ratings.AccountIDOf(string(guest.Account))
		require.NoError(t, err)
		assert.Equal(t, []ratings.Answer{{
			Question: "capital:ee", Account: account, Correct: correct, After: 3 * time.Second, At: now,
		}}, answers.answers, "correct: %v", correct)
	}
}

func TestATimeoutIsCountedWrongAndTimedOut(t *testing.T) {
	useCase, _, _, answers := newCountedUseCase(bonuses.Answered{Question: "capital:ee", TimedOut: true}, true)

	out, err := useCase.Execute(t.Context(), answerOf("t", 3, ""))
	require.NoError(t, err)

	assert.False(t, out.Correct)
	require.Len(t, answers.answers, 1)
	assert.True(t, answers.answers[0].TimedOut)
	assert.False(t, answers.answers[0].Correct)
}

func TestACallerWithNoAccountIsCountedWithNone(t *testing.T) {
	registry := &fakeRegistry{answered: bonuses.Answered{Question: "capital:ee"}, known: true}
	answers := &counted{}
	useCase := answer_quiz_usecase.New(registry, bonuses.NewFakeWallet(), answers, cpredis.NewRecordedEvents(),
		cptime.NewFixedClock(now))

	_, err := useCase.Execute(t.Context(), answer_quiz_usecase.In{Caller: bonuses.Caller{Scope: "1.2.3.4"}, Token: "t"})
	require.NoError(t, err)

	require.Len(t, answers.answers, 1)
	assert.Equal(t, ratings.NoAccount, answers.answers[0].Account, "the question counts it; nobody is named")
}

func TestALateAnswerIsCountedAndTheQuizIsStillGone(t *testing.T) {
	useCase, registry, wallet, answers := newCountedUseCase(bonuses.Answered{
		Question: "capital:ee", TimedOut: true, Late: true, CorrectChoice: 2,
	}, true)

	_, err := useCase.Execute(t.Context(), answerOf("t", 3, ""))

	require.ErrorIs(t, err, answer_quiz_usecase.ErrNoSuchQuiz, "to the player the quiz lapsed, as it did before")
	require.Len(t, answers.answers, 1)
	assert.False(t, answers.answers[0].Correct)
	assert.Empty(t, registry.published)
	assert.Empty(t, wallet.Payments())
}

func TestAQuizNotFoundIsNotCounted(t *testing.T) {
	useCase, _, _, answers := newCountedUseCase(bonuses.Answered{}, false)

	_, err := useCase.Execute(t.Context(), answerOf("t", 0, ""))

	require.ErrorIs(t, err, answer_quiz_usecase.ErrNoSuchQuiz)
	assert.Empty(t, answers.answers)
}

func TestAnAnswerIsCountedWhenTheCallerLeavesAfterIt(t *testing.T) {
	useCase, _, _, answers := newCountedUseCase(bonuses.Answered{Question: "capital:ee"}, true)
	ctx, leave := context.WithCancel(t.Context())
	leave()

	_, _ = useCase.Execute(ctx, answerOf("t", 0, ""))

	assert.Equal(t, []error{nil}, answers.gone, "the count runs on a context the caller's leaving does not end")
}

func TestACountThatFailsCostsThePlayerNothing(t *testing.T) {
	useCase, _, store, answers := newCountedUseCase(rightFor(bonuses.KindBomb, 1), true)
	answers.err = errors.New("postgres is down")

	out, err := useCase.Execute(t.Context(), answerOf("t", 0, ""))

	require.NoError(t, err)
	assert.True(t, out.Correct)
	assert.Equal(t, bonuses.Hand{Bomb: true}, store.Kept(guest.Account))
}
