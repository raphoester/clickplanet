package post_teaser_usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/teasers"
	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/teasers/usecases/post_teaser_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var now = time.Date(2026, 11, 13, 7, 43, 0, 0, time.UTC)

type battles []teasers.Teaser

func (b battles) Teasers(context.Context, time.Duration) ([]teasers.Teaser, error) {
	return b, nil
}

type posts struct {
	teased []teasers.BattleID
	err    error
	since  time.Time
}

func (p *posts) Teased(_ context.Context, since time.Time) ([]teasers.BattleID, error) {
	p.since = since
	return p.teased, p.err
}

type poster struct{ posted []teasers.Post }

func (p *poster) Post(_ context.Context, post teasers.Post) (teasers.Outcome, error) {
	p.posted = append(p.posted, post)
	return teasers.Outcome{Result: teasers.Posted, Post: "post-1"}, nil
}

func ringWar() teasers.Teaser {
	return teasers.Teaser{Battle: "entry-7", StartsAt: now.Add(32 * time.Hour), Cards: []teasers.Card{{Headline: "Gain the most land"}}}
}

var media = teasers.Media{Photos: []teasers.MediaURL{"https://media.example/card-1.jpg"}, Video: "https://media.example/teaser.mp4"}

func execute(t *testing.T, listed battles, read *posts, battle teasers.BattleID, files teasers.Media) (teasers.Outcome, *poster, error) {
	t.Helper()
	made := &poster{}
	outcome, err := post_teaser_usecase.New(listed, read, made, cptime.NewFixedClock(now)).Execute(t.Context(), battle, files)
	return outcome, made, err
}

func TestABattleStillDueIsPostedWithItsMedia(t *testing.T) {
	read := &posts{teased: []teasers.BattleID{"entry-6"}}

	outcome, made, err := execute(t, battles{ringWar()}, read, "entry-7", media)

	require.NoError(t, err)
	assert.Equal(t, teasers.Posted, outcome.Result)
	require.Len(t, made.posted, 1)
	assert.Equal(t, ringWar(), made.posted[0].Teaser())
	assert.Equal(t, media, made.posted[0].Media())
	assert.Equal(t, teasers.TeasedSince(now), read.since)
}

func TestABattleNoLongerDueIsSkipped(t *testing.T) {
	outcome, made, err := execute(t, battles{ringWar()}, &posts{teased: []teasers.BattleID{"entry-7"}}, "entry-7", media)
	require.NoError(t, err)
	assert.Equal(t, teasers.Outcome{Result: teasers.Skipped, Why: "teased already"}, outcome)
	assert.Empty(t, made.posted)

	outcome, _, err = execute(t, battles{ringWar()}, &posts{}, "entry-9", media)
	require.NoError(t, err)
	assert.Equal(t, teasers.Outcome{Result: teasers.Skipped, Why: "no longer on the schedule within 36 hours"}, outcome)
}

func TestMediaThatAreNotTheBattlesAreRefused(t *testing.T) {
	_, made, err := execute(t, battles{ringWar()}, &posts{}, "entry-7", teasers.Media{Video: "https://media.example/teaser.mp4"})

	require.ErrorIs(t, err, teasers.ErrInvalidPost)
	assert.Empty(t, made.posted)
}

func TestPostsPublishingCannotListAreBusyAndAnyOtherFailureIsAnError(t *testing.T) {
	outcome, _, err := execute(t, battles{ringWar()}, &posts{err: &teasers.BusyError{Reason: "restarting"}}, "entry-7", media)
	require.NoError(t, err)
	assert.Equal(t, teasers.Busy, outcome.Result)

	off := errors.New("publishing is off")
	_, _, err = execute(t, battles{ringWar()}, &posts{err: off}, "entry-7", media)
	require.ErrorIs(t, err, off)
}
