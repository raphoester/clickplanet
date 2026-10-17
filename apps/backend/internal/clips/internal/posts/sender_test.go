package posts_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/posts"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var (
	now     = time.Date(2026, 10, 12, 12, 0, 0, 0, time.UTC)
	backoff = posts.Backoff{time.Minute, 5 * time.Minute, 15 * time.Minute}
)

func busy(retryAfter time.Duration) error {
	return &posts.BusyError{RetryAfter: retryAfter, Reason: "rate limit exceeded"}
}

type sending struct {
	service *posts.ScriptedService
	pause   *posts.RecordedPause
	post    posts.Post
	sender  *posts.Sender
}

func send(t *testing.T, refusals ...error) sending {
	t.Helper()

	clock := cptime.NewFixedClock(now)
	service := posts.NewScriptedService(refusals...)
	pause := posts.NewRecordedPause(clock)
	return sending{
		service: service, pause: pause,
		post:   plan.Posts([]posts.Clip{clip(t, 1, "BEST")}, now, posts.LoadedDice(0))[0],
		sender: posts.NewSender(service, backoff, pause, clock),
	}
}

func TestAPostTheServiceTakesIsSentOnce(t *testing.T) {
	s := send(t)

	scheduled, err := s.sender.Send(t.Context(), s.post)

	require.NoError(t, err)
	assert.Equal(t, posts.Scheduled{ID: "post-BEST"}, scheduled)
	assert.Equal(t, []posts.PostSnapshot{s.post.TakeSnapshot()}, s.service.Scheduled)
	assert.Empty(t, s.pause.Waited)
}

func TestABusyServiceIsTriedAgainAfterALongerWaitEachTime(t *testing.T) {
	s := send(t, busy(0), busy(0), busy(0))

	scheduled, err := s.sender.Send(t.Context(), s.post)

	require.NoError(t, err)
	assert.Equal(t, posts.PostID("post-BEST"), scheduled.ID)
	assert.Equal(t, []time.Duration{time.Minute, 5 * time.Minute, 15 * time.Minute}, s.pause.Waited)
}

func TestAServiceThatSaysHowLongToWaitIsWaitedForThatLong(t *testing.T) {
	s := send(t, busy(3*time.Minute), busy(time.Minute))

	_, err := s.sender.Send(t.Context(), s.post)

	require.NoError(t, err)
	assert.Equal(t, []time.Duration{3 * time.Minute, 5 * time.Minute}, s.pause.Waited)
}

func TestAServiceStillBusyAfterTheLastWaitFailsThePost(t *testing.T) {
	s := send(t, busy(0), busy(0), busy(0), busy(0))

	_, err := s.sender.Send(t.Context(), s.post)

	var refusal *posts.BusyError
	require.ErrorAs(t, err, &refusal)
	assert.Empty(t, s.service.Scheduled)
	assert.Len(t, s.pause.Waited, 3)
}

func TestAServiceBusyForLongerThanTheLongestWaitFailsThePostAtOnce(t *testing.T) {
	s := send(t, busy(6*time.Hour))

	_, err := s.sender.Send(t.Context(), s.post)

	var refusal *posts.BusyError
	require.ErrorAs(t, err, &refusal)
	assert.Empty(t, s.pause.Waited)
}

func TestAPostOnSomeOfItsPlatformsIsNotTriedAgain(t *testing.T) {
	s := send(t)
	s.service.Refusing = map[posts.Platform]error{posts.YouTube: errors.New("no youtube account")}

	scheduled, err := s.sender.Send(t.Context(), s.post)

	require.NoError(t, err)
	assert.Equal(t, s.service.Refusing, scheduled.Refused)
	assert.Len(t, s.service.Scheduled, 1)
	assert.Empty(t, s.pause.Waited)
}

func TestAnyOtherRefusalIsNotTriedAgain(t *testing.T) {
	refused := errors.New("the caption is too long")
	s := send(t, refused)

	_, err := s.sender.Send(t.Context(), s.post)

	require.ErrorIs(t, err, refused)
	assert.Empty(t, s.pause.Waited)
	assert.Empty(t, s.service.Scheduled)
}

func TestAPostTooCloseToItsTimeIsNeverSent(t *testing.T) {
	clock := cptime.NewFixedClock(now.Add(3*time.Hour + time.Minute))
	service := posts.NewScriptedService()
	post := plan.Posts([]posts.Clip{clip(t, 1, "BEST")}, now, posts.LoadedDice(0))[0]

	_, err := posts.NewSender(service, backoff, posts.NewRecordedPause(clock), clock).Send(t.Context(), post)

	require.ErrorIs(t, err, posts.ErrNoTimeToReview)
	assert.Empty(t, service.Scheduled)
}

func TestAWaitThatEatsTheReviewStopsTheTries(t *testing.T) {
	clock := cptime.NewFixedClock(now.Add(3*time.Hour - 30*time.Second))
	service := posts.NewScriptedService(busy(0))
	post := plan.Posts([]posts.Clip{clip(t, 1, "BEST")}, now, posts.LoadedDice(0))[0]

	_, err := posts.NewSender(service, backoff, posts.NewRecordedPause(clock), clock).Send(t.Context(), post)

	require.ErrorIs(t, err, posts.ErrNoTimeToReview)
	assert.Empty(t, service.Scheduled)
}
