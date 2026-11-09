package teasers_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/teasers"
)

var (
	both    = []teasers.Platform{teasers.TikTok, teasers.YouTube}
	tiktok  = []teasers.Platform{teasers.TikTok}
	youtube = []teasers.Platform{teasers.YouTube}
	slotAt  = time.Date(2026, 11, 13, 14, 0, 0, 0, time.UTC)
)

type asked struct {
	platforms []teasers.Platform
	notBefore time.Time
	by        time.Time
}

// A calendar that gives each slot of slots in turn, a found false for a zero one, and a service that answers each of
// answers in turn.
type fakePublishing struct {
	slots   []teasers.Slot
	planErr error
	answers []error
	asked   []asked
	posted  []teasers.Post
}

func (f *fakePublishing) Slot(_ context.Context, platforms []teasers.Platform, notBefore, by time.Time) (teasers.Slot, bool, error) {
	f.asked = append(f.asked, asked{platforms: platforms, notBefore: notBefore, by: by})
	if f.planErr != nil {
		return teasers.Slot{}, false, f.planErr
	}
	slot := f.slots[0]
	f.slots = f.slots[1:]
	return slot, !slot.At.IsZero(), nil
}

func (f *fakePublishing) Schedule(_ context.Context, post teasers.Post) (teasers.Receipt, error) {
	f.posted = append(f.posted, post)
	err := f.answers[0]
	f.answers = f.answers[1:]
	if err != nil {
		return teasers.Receipt{}, err
	}
	return teasers.Receipt{ID: "post-1"}, nil
}

func postWith(t *testing.T, publishing *fakePublishing) (teasers.Outcome, error) {
	t.Helper()
	return teasers.NewPoster(publishing).Post(t.Context(), post(t, ringWar()))
}

func TestATeaserGoesOutOnBothPlatformsAtTheSlotByItsBattle(t *testing.T) {
	publishing := &fakePublishing{slots: []teasers.Slot{{At: slotAt, Platforms: both}}, answers: []error{nil}}

	outcome, err := postWith(t, publishing)

	require.NoError(t, err)
	assert.Equal(t, teasers.Posted, outcome.Result)
	assert.Equal(t, "post-1", outcome.Post)
	assert.Equal(t, both, outcome.Platforms)
	notBefore, by := ringWar().Window()
	assert.Equal(t, []asked{{platforms: both, notBefore: notBefore, by: by}}, publishing.asked)
	assert.Equal(t, teasers.Slot{At: slotAt, Platforms: both}, publishing.posted[0].Slot())
}

func TestATeaserThatBothCannotTakeGoesOnTikTokAloneElseYouTubeAlone(t *testing.T) {
	publishing := &fakePublishing{slots: []teasers.Slot{{}, {}, {At: slotAt, Platforms: youtube}}, answers: []error{nil}}
	outcome, err := postWith(t, publishing)
	require.NoError(t, err)
	assert.Equal(t, youtube, outcome.Platforms)
	assert.Equal(t, [][]teasers.Platform{both, tiktok, youtube},
		[][]teasers.Platform{publishing.asked[0].platforms, publishing.asked[1].platforms, publishing.asked[2].platforms})

	publishing = &fakePublishing{slots: []teasers.Slot{{}, {At: slotAt, Platforms: tiktok}}, answers: []error{nil}}
	outcome, err = postWith(t, publishing)
	require.NoError(t, err)
	assert.Equal(t, tiktok, outcome.Platforms)
}

func TestATeaserNoPlatformTakesBeforeItsBattleIsHeld(t *testing.T) {
	publishing := &fakePublishing{slots: []teasers.Slot{{}, {}, {}}}

	outcome, err := postWith(t, publishing)

	require.NoError(t, err)
	assert.Equal(t, teasers.Held, outcome.Result)
	assert.Empty(t, publishing.posted)
}

func TestABusyServiceIsTheCallersToWaitFor(t *testing.T) {
	busy := &teasers.BusyError{RetryAfter: 2 * time.Minute, Reason: "zernio answered 429"}

	outcome, err := postWith(t, &fakePublishing{slots: []teasers.Slot{{At: slotAt, Platforms: both}}, answers: []error{busy}})
	require.NoError(t, err)
	assert.Equal(t, teasers.Outcome{Result: teasers.Busy, Why: "zernio answered 429", RetryAfter: 2 * time.Minute}, outcome)

	outcome, err = postWith(t, &fakePublishing{planErr: &teasers.BusyError{Reason: "publishing is restarting"}})
	require.NoError(t, err)
	assert.Equal(t, teasers.Busy, outcome.Result, "a plan the module cannot give is busy too")
}

func TestATakenSlotIsPlannedAgainThreeTimesThenBusy(t *testing.T) {
	taken := teasers.ErrTaken
	slots := []teasers.Slot{{At: slotAt, Platforms: both}, {At: slotAt, Platforms: both}, {At: slotAt, Platforms: both}, {At: slotAt, Platforms: both}}

	outcome, err := postWith(t, &fakePublishing{slots: slots, answers: []error{taken, taken, nil}})
	require.NoError(t, err)
	assert.Equal(t, teasers.Posted, outcome.Result)

	publishing := &fakePublishing{slots: slots, answers: []error{taken, taken, taken, taken}}
	outcome, err = postWith(t, publishing)
	require.NoError(t, err)
	assert.Equal(t, teasers.Busy, outcome.Result)
	assert.Len(t, publishing.posted, 4)
}

func TestARefusalForGoodFailsThePostAndAnyOtherErrorIsReturned(t *testing.T) {
	outcome, err := postWith(t, &fakePublishing{
		slots: []teasers.Slot{{At: slotAt, Platforms: both}}, answers: []error{&teasers.RefusedError{Reason: "the caption is refused"}},
	})
	require.NoError(t, err)
	assert.Equal(t, teasers.Outcome{Result: teasers.Failed, Why: "the caption is refused"}, outcome)

	off := errors.New("publishing is off")
	_, err = postWith(t, &fakePublishing{planErr: off})
	require.ErrorIs(t, err, off)
}
