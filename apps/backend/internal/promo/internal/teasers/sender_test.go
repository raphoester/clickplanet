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
	answers []error
	asked   []asked
	posted  []teasers.Post
}

func (f *fakePublishing) Slot(_ context.Context, platforms []teasers.Platform, notBefore, by time.Time) (teasers.Slot, bool, error) {
	f.asked = append(f.asked, asked{platforms: platforms, notBefore: notBefore, by: by})
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

type pauses []time.Duration

func (p *pauses) Wait(_ context.Context, d time.Duration) error {
	*p = append(*p, d)
	return nil
}

var backoff = teasers.Backoff{time.Minute, 5 * time.Minute, 15 * time.Minute}

func send(t *testing.T, publishing *fakePublishing) (teasers.Post, teasers.Receipt, bool, pauses, error) {
	t.Helper()
	waited := pauses{}
	post, receipt, held, err := teasers.NewSender(publishing, backoff, &waited).Send(t.Context(), teasers.Post{Teaser: ringWar()})
	return post, receipt, held, waited, err
}

func TestATeaserGoesOutOnBothPlatformsAtTheSlotByItsBattle(t *testing.T) {
	at := time.Date(2026, 11, 13, 14, 0, 0, 0, time.UTC)
	publishing := &fakePublishing{slots: []teasers.Slot{{At: at, Platforms: both}}, answers: []error{nil}}

	post, receipt, held, _, err := send(t, publishing)

	require.NoError(t, err)
	assert.False(t, held)
	assert.Equal(t, "post-1", receipt.ID)
	assert.Equal(t, teasers.Slot{At: at, Platforms: both}, post.Slot)
	notBefore, by := ringWar().Window()
	assert.Equal(t, []asked{{platforms: both, notBefore: notBefore, by: by}}, publishing.asked)
}

func TestATeaserThatBothCannotTakeGoesOnTikTokAloneElseYouTubeAlone(t *testing.T) {
	at := time.Date(2026, 11, 14, 8, 0, 0, 0, time.UTC)

	publishing := &fakePublishing{slots: []teasers.Slot{{}, {}, {At: at, Platforms: youtube}}, answers: []error{nil}}
	post, _, held, _, err := send(t, publishing)
	require.NoError(t, err)
	assert.False(t, held)
	assert.Equal(t, youtube, post.Slot.Platforms)
	assert.Equal(t, [][]teasers.Platform{both, tiktok, youtube},
		[][]teasers.Platform{publishing.asked[0].platforms, publishing.asked[1].platforms, publishing.asked[2].platforms})

	publishing = &fakePublishing{slots: []teasers.Slot{{}, {At: at, Platforms: tiktok}}, answers: []error{nil}}
	post, _, _, _, err = send(t, publishing)
	require.NoError(t, err)
	assert.Equal(t, tiktok, post.Slot.Platforms)
}

func TestATeaserNoPlatformTakesBeforeItsBattleIsHeld(t *testing.T) {
	publishing := &fakePublishing{slots: []teasers.Slot{{}, {}, {}}}

	_, _, held, _, err := send(t, publishing)

	require.NoError(t, err)
	assert.True(t, held)
	assert.Empty(t, publishing.posted)
}

func TestABusyServiceIsWaitedForAndTheSlotPlannedAgain(t *testing.T) {
	first, second := time.Date(2026, 11, 13, 14, 0, 0, 0, time.UTC), time.Date(2026, 11, 13, 15, 0, 0, 0, time.UTC)
	publishing := &fakePublishing{
		slots:   []teasers.Slot{{At: first, Platforms: both}, {At: second, Platforms: both}},
		answers: []error{&teasers.BusyError{RetryAfter: 2 * time.Minute, Reason: "429"}, nil},
	}

	post, _, _, waited, err := send(t, publishing)

	require.NoError(t, err)
	assert.Equal(t, second, post.Slot.At)
	assert.Equal(t, pauses{2 * time.Minute}, waited)
}

func TestAServiceBusyPastTheBackoffFailsThePost(t *testing.T) {
	at := time.Date(2026, 11, 13, 14, 0, 0, 0, time.UTC)
	busy := &teasers.BusyError{Reason: "503"}
	publishing := &fakePublishing{
		slots:   []teasers.Slot{{At: at, Platforms: both}, {At: at, Platforms: both}, {At: at, Platforms: both}, {At: at, Platforms: both}},
		answers: []error{busy, busy, busy, busy},
	}

	_, _, _, waited, err := send(t, publishing)

	require.ErrorIs(t, err, busy)
	assert.Equal(t, pauses(backoff), waited)
}

func TestATakenSlotIsPlannedAgainThreeTimesAtTheMost(t *testing.T) {
	at := time.Date(2026, 11, 13, 14, 0, 0, 0, time.UTC)
	taken := teasers.ErrTaken
	slots := []teasers.Slot{{At: at, Platforms: both}, {At: at, Platforms: both}, {At: at, Platforms: both}, {At: at, Platforms: both}}

	publishing := &fakePublishing{slots: slots, answers: []error{taken, taken, nil}}
	_, receipt, _, _, err := send(t, publishing)
	require.NoError(t, err)
	assert.Equal(t, "post-1", receipt.ID)

	publishing = &fakePublishing{slots: slots, answers: []error{taken, taken, taken, taken}}
	_, _, _, _, err = send(t, publishing)
	require.ErrorIs(t, err, teasers.ErrTaken)
	assert.Len(t, publishing.posted, 4)
}

func TestARefusalForGoodFailsThePostAtOnce(t *testing.T) {
	at := time.Date(2026, 11, 13, 14, 0, 0, 0, time.UTC)
	refused := errors.New("the caption is refused")
	publishing := &fakePublishing{slots: []teasers.Slot{{At: at, Platforms: both}}, answers: []error{refused}}

	_, _, _, waited, err := send(t, publishing)

	require.ErrorIs(t, err, refused)
	assert.Empty(t, waited)
}
