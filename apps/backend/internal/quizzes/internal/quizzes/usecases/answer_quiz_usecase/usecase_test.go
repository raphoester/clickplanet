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

	quizzesv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/quizzes/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/quizzes/internal/quizzes"
	"github.com/raphoester/clickplanet.lol-backend/internal/quizzes/internal/quizzes/usecases/answer_quiz_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/quizzes/internal/ratings"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var now = time.Date(2026, 10, 25, 12, 0, 0, 0, time.UTC)

var guest = quizzes.Caller{Scope: "1.2.3.4", Account: "0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11"}

func answerOf(token string, choice int, country string) answer_quiz_usecase.In {
	return answer_quiz_usecase.In{Caller: guest, Token: token, Choice: choice, CountryID: country}
}

type fakeRegistry struct {
	answered quizzes.Answered
	known    bool

	choice     int
	forEntrant quizzes.Entrant
	published  []quizzes.Won
}

func (f *fakeRegistry) AnswerQuiz(_ string, entrant quizzes.Entrant, choice int) (quizzes.Answered, bool) {
	f.choice, f.forEntrant = choice, entrant

	return f.answered, f.known
}

func (f *fakeRegistry) Publish(won quizzes.Won) {
	f.published = append(f.published, won)
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

// goneOutbox is an outbox that keeps the context's error at each record, and fails when told to.
type goneOutbox struct {
	cpatomic.FakeOutbox
	gone    []error
	failing error
}

func (o *goneOutbox) Record(ctx context.Context, events ...proto.Message) error {
	o.gone = append(o.gone, ctx.Err())
	if o.failing != nil {
		return o.failing
	}
	return o.FakeOutbox.Record(ctx, events...) //nolint:wrapcheck // the fake's own.
}

type fixture struct {
	useCase  *answer_quiz_usecase.UseCase
	registry *fakeRegistry
	answers  *counted
	outbox   *goneOutbox
}

func newUseCase(answered quizzes.Answered, known bool) fixture {
	f := fixture{
		registry: &fakeRegistry{answered: answered, known: known},
		answers:  &counted{},
		outbox:   &goneOutbox{},
	}
	f.useCase = answer_quiz_usecase.New(f.registry, f.answers, cpatomic.FakeTransactioner{}, f.outbox,
		cptime.NewFixedClock(now))
	return f
}

func TestARightAnswerIsToldAndAnnouncedAndPaysNothingHere(t *testing.T) {
	f := newUseCase(quizzes.Answered{Outcome: ratings.Right, CorrectChoice: 2, Subject: "ee"}, true)

	out, err := f.useCase.Execute(t.Context(), answerOf("t", 2, "bg"))
	require.NoError(t, err)

	assert.Equal(t, answer_quiz_usecase.Out{Correct: true, CorrectChoice: 2}, out)
	assert.Equal(t, []quizzes.Won{{CountryID: "bg", Subject: "ee"}}, f.registry.published,
		"everyone hears who won it, and what the question was about")
	assert.Equal(t, quizzes.Entrant("1.2.3.4"), f.registry.forEntrant, "a guest's answer is settled against its scope")
}

func TestAWrongAnswerSaysTheRightChoiceAndIsAnnouncedToNobody(t *testing.T) {
	f := newUseCase(quizzes.Answered{Outcome: ratings.Wrong, CorrectChoice: 1}, true)

	out, err := f.useCase.Execute(t.Context(), answerOf("t", 0, "bg"))
	require.NoError(t, err, "a wrong answer landed; it is not a failed call")

	assert.Equal(t, answer_quiz_usecase.Out{CorrectChoice: 1}, out, "it still says which one was right")
	assert.Empty(t, f.registry.published)
}

func TestEveryAnswerIsToldRightOrWrongNamedByItsToken(t *testing.T) {
	for _, outcome := range []ratings.Outcome{ratings.Right, ratings.Wrong, ratings.TimedOut} {
		f := newUseCase(quizzes.Answered{Outcome: outcome, Welcome: true}, true)

		_, err := f.useCase.Execute(t.Context(), answerOf("a-token", 0, ""))
		require.NoError(t, err)

		recorded := f.outbox.Recorded()
		require.Len(t, recorded, 1)
		assert.True(t, proto.Equal(&quizzesv1.QuizAnswered{
			AccountId: string(guest.Account), Correct: outcome == ratings.Right, AnsweredAt: timestamppb.New(now),
			Token: "a-token", Welcome: true,
		}, recorded[0]), "outcome: %v", outcome)
	}
}

func TestACallerWithNoAccountIsToldOfToNobody(t *testing.T) {
	f := newUseCase(quizzes.Answered{Outcome: ratings.Right}, true)

	_, err := f.useCase.Execute(t.Context(), answer_quiz_usecase.In{Caller: quizzes.Caller{Scope: "1.2.3.4"}, Token: "t"})
	require.NoError(t, err)

	assert.Empty(t, f.outbox.Recorded(), "nobody is paid for it")
	require.Len(t, f.answers.answers, 1)
	assert.Equal(t, ratings.NoAccount, f.answers.answers[0].Account, "the question counts it; nobody is named")
}

func TestAQuizThatIsNotThisCallersIsNotFound(t *testing.T) {
	f := newUseCase(quizzes.Answered{}, false)

	_, err := f.useCase.Execute(t.Context(), answerOf("t", 0, ""))
	require.ErrorIs(t, err, answer_quiz_usecase.ErrNoSuchQuiz)
	assert.Empty(t, f.registry.published)
	assert.Empty(t, f.outbox.Recorded())
	assert.Empty(t, f.answers.answers, "a quiz not found is not counted")
}

func TestAnAnswerIsToldWhenTheCallerLeavesAfterIt(t *testing.T) {
	f := newUseCase(quizzes.Answered{Outcome: ratings.Right}, true)
	ctx, leave := context.WithCancel(t.Context())
	leave()

	_, err := f.useCase.Execute(ctx, answerOf("t", 0, ""))
	require.NoError(t, err)

	assert.Equal(t, []error{nil}, f.outbox.gone, "the tell runs on a context the caller's leaving does not end")
	assert.Equal(t, []error{nil}, f.answers.gone, "and so does the count")
	assert.Len(t, f.outbox.Recorded(), 1)
}

func TestAnAnswerNotToldIsAFailureAndIsAnnouncedToNobody(t *testing.T) {
	f := newUseCase(quizzes.Answered{Outcome: ratings.Right}, true)
	f.outbox.failing = errors.New("postgres is down")

	_, err := f.useCase.Execute(t.Context(), answerOf("t", 0, ""))

	require.Error(t, err)
	assert.NotErrorIs(t, err, answer_quiz_usecase.ErrNoSuchQuiz)
	assert.Empty(t, f.registry.published)
}

func TestEveryAnswerIsCountedForItsQuestionWithItsOutcome(t *testing.T) {
	for _, outcome := range []ratings.Outcome{ratings.Right, ratings.Wrong, ratings.TimedOut} {
		f := newUseCase(quizzes.Answered{Outcome: outcome, Question: "capital:ee", After: 3 * time.Second}, true)

		_, err := f.useCase.Execute(t.Context(), answerOf("t", 0, ""))
		require.NoError(t, err)

		account, err := ratings.AccountIDOf(string(guest.Account))
		require.NoError(t, err)
		assert.Equal(t, []ratings.Answer{{
			Question: "capital:ee", Account: account, Outcome: outcome, After: 3 * time.Second, At: now,
		}}, f.answers.answers, "outcome: %v", outcome)
	}
}

func TestALateAnswerIsCountedAndTheQuizIsStillGone(t *testing.T) {
	f := newUseCase(quizzes.Answered{Question: "capital:ee", Outcome: ratings.TimedOut, Late: true, CorrectChoice: 2}, true)

	_, err := f.useCase.Execute(t.Context(), answerOf("t", 3, ""))

	require.ErrorIs(t, err, answer_quiz_usecase.ErrNoSuchQuiz, "to the player the quiz lapsed, as it did before")
	require.Len(t, f.answers.answers, 1)
	assert.Equal(t, ratings.TimedOut, f.answers.answers[0].Outcome)
	assert.Empty(t, f.registry.published)
	assert.Empty(t, f.outbox.Recorded(), "a late answer is no answer to the other modules")
}

func TestACountThatFailsCostsThePlayerNothing(t *testing.T) {
	f := newUseCase(quizzes.Answered{Outcome: ratings.Right}, true)
	f.answers.err = errors.New("postgres is down")

	out, err := f.useCase.Execute(t.Context(), answerOf("t", 0, ""))

	require.NoError(t, err)
	assert.True(t, out.Correct)
	assert.Len(t, f.outbox.Recorded(), 1, "the right answer is still told, and paid")
}
