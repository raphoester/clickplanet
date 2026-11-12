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
	first   = posts.Slot{At: time.Date(2026, 10, 12, 12, 5, 0, 0, time.UTC), Platforms: both}
	later   = posts.Slot{At: time.Date(2026, 10, 12, 12, 21, 0, 0, time.UTC), Platforms: []posts.Platform{posts.YouTube}}
)

func busy(retryAfter time.Duration) error {
	return &posts.BusyError{RetryAfter: retryAfter, Reason: "rate limit exceeded"}
}

type sending struct {
	planner *posts.ScriptedPlanner
	service *posts.ScriptedService
	pause   *posts.RecordedPause
	post    posts.Post
	sender  *posts.Sender
}

// A calendar that gives the first slot, then the later one, and a service that answers the refusals in turn.
func send(t *testing.T, refusals ...error) sending {
	t.Helper()

	planner := &posts.ScriptedPlanner{Slots: []posts.Slot{first, later}}
	service := posts.NewScriptedService(refusals...)
	pause := posts.NewRecordedPause(cptime.NewFixedClock(now))
	return sending{
		planner: planner, service: service, pause: pause,
		post:   asap.Post(clip(t, 1, "BEST"), 1, 1, posts.LoadedDice(0)),
		sender: posts.NewSender(planner, service, backoff, pause),
	}
}

func TestAPostTheServiceTakesIsSentOnceAtTheSlotTheCalendarGave(t *testing.T) {
	s := send(t)

	sent, scheduled, held, err := s.sender.Send(t.Context(), s.post)

	require.NoError(t, err)
	assert.False(t, held)
	assert.Equal(t, posts.Scheduled{ID: "post-BEST"}, scheduled)
	assert.Equal(t, first.At, sent.TakeSnapshot().At)
	assert.Equal(t, []posts.PostSnapshot{sent.TakeSnapshot()}, s.service.Scheduled)
	assert.Equal(t, [][]posts.Platform{both}, s.planner.Asked, "the platforms of its place")
	assert.Empty(t, s.pause.Waited)
}

func TestABusyServiceIsTriedAgainAfterALongerWaitEachTimeAtASlotPlannedAgain(t *testing.T) {
	s := send(t, busy(0), busy(0), busy(0))

	sent, scheduled, _, err := s.sender.Send(t.Context(), s.post)

	require.NoError(t, err)
	assert.Equal(t, posts.PostID("post-BEST"), scheduled.ID)
	assert.Equal(t, []time.Duration{time.Minute, 5 * time.Minute, 15 * time.Minute}, s.pause.Waited)
	assert.Len(t, s.planner.Asked, 4, "a wait never sends it at a time already past or crowded")
	assert.Equal(t, later.At, sent.TakeSnapshot().At)
	assert.Equal(t, later.Platforms, sent.TakeSnapshot().Platforms)
}

func TestAServiceThatSaysHowLongToWaitIsWaitedForThatLong(t *testing.T) {
	s := send(t, busy(3*time.Minute), busy(time.Minute))

	_, _, _, err := s.sender.Send(t.Context(), s.post)

	require.NoError(t, err)
	assert.Equal(t, []time.Duration{3 * time.Minute, 5 * time.Minute}, s.pause.Waited)
}

func TestAServiceStillBusyAfterTheLastWaitFailsThePost(t *testing.T) {
	s := send(t, busy(0), busy(0), busy(0), busy(0))

	_, _, held, err := s.sender.Send(t.Context(), s.post)

	var refusal *posts.BusyError
	require.ErrorAs(t, err, &refusal)
	assert.False(t, held)
	assert.Empty(t, s.service.Scheduled)
	assert.Len(t, s.pause.Waited, 3)
}

func TestAServiceBusyForLongerThanTheLongestWaitFailsThePostAtOnce(t *testing.T) {
	s := send(t, busy(6*time.Hour))

	_, _, _, err := s.sender.Send(t.Context(), s.post)

	var refusal *posts.BusyError
	require.ErrorAs(t, err, &refusal)
	assert.Empty(t, s.pause.Waited)
}

func TestASlotTakenIsPlannedAgainAtOnceAndTakenTooOftenFailsThePost(t *testing.T) {
	s := send(t, posts.ErrTaken)

	sent, scheduled, _, err := s.sender.Send(t.Context(), s.post)

	require.NoError(t, err)
	assert.Equal(t, posts.PostID("post-BEST"), scheduled.ID)
	assert.Equal(t, later.At, sent.TakeSnapshot().At)
	assert.Empty(t, s.pause.Waited)

	s = send(t, posts.ErrTaken, posts.ErrTaken, posts.ErrTaken, posts.ErrTaken)
	_, _, _, err = s.sender.Send(t.Context(), s.post)
	require.ErrorIs(t, err, posts.ErrTaken)
	assert.Len(t, s.planner.Asked, 4)
}

func TestAPostOnSomeOfItsPlatformsIsNotTriedAgain(t *testing.T) {
	s := send(t)
	s.service.Refusing = map[posts.Platform]error{posts.YouTube: errors.New("no youtube account")}

	_, scheduled, _, err := s.sender.Send(t.Context(), s.post)

	require.NoError(t, err)
	assert.Equal(t, s.service.Refusing, scheduled.Refused)
	assert.Len(t, s.service.Scheduled, 1)
	assert.Empty(t, s.pause.Waited)
}

func TestAnyOtherRefusalIsNotTriedAgain(t *testing.T) {
	refused := errors.New("the caption is too long")
	s := send(t, refused)

	_, _, _, err := s.sender.Send(t.Context(), s.post)

	require.ErrorIs(t, err, refused)
	assert.Empty(t, s.pause.Waited)
	assert.Empty(t, s.service.Scheduled)
}

func TestABusyCalendarIsWaitedForAsTheServiceIs(t *testing.T) {
	s := send(t)
	s.planner.Failed = busy(0)

	_, _, _, err := s.sender.Send(t.Context(), s.post)

	var refusal *posts.BusyError
	require.ErrorAs(t, err, &refusal)
	assert.Len(t, s.pause.Waited, 3)
	assert.Len(t, s.planner.Asked, 4)
}
