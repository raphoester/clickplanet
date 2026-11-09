package post_teasers_usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/teasers"
	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/teasers/usecases/post_teasers_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var (
	now  = time.Date(2026, 11, 13, 7, 43, 0, 0, time.UTC)
	both = []teasers.Platform{teasers.TikTok, teasers.YouTube}
)

type fakeBackend struct {
	listed []teasers.Teaser
	teased []teasers.BattleID
}

func (f fakeBackend) Teasers(context.Context, time.Duration) ([]teasers.Teaser, error) {
	return f.listed, nil
}

func (f fakeBackend) Teased(context.Context, time.Time) ([]teasers.BattleID, error) {
	return f.teased, nil
}

type renders []teasers.Rendered

func (r renders) Rendered(context.Context) ([]teasers.Rendered, error) {
	return r, nil
}

// Sends each post at its battle's start less a day; held when a battle is in held, failed when in failing.
type fakeSender struct {
	held    []teasers.BattleID
	failing []teasers.BattleID
	sent    []teasers.Post
}

func (f *fakeSender) Send(_ context.Context, post teasers.Post) (teasers.Post, teasers.Receipt, bool, error) {
	f.sent = append(f.sent, post)
	for _, battle := range f.held {
		if battle == post.Teaser.Battle {
			return post, teasers.Receipt{}, true, nil
		}
	}
	for _, battle := range f.failing {
		if battle == post.Teaser.Battle {
			return post, teasers.Receipt{}, false, errors.New("the caption is refused")
		}
	}
	post.Slot = teasers.Slot{At: post.Teaser.StartsAt.Add(-24 * time.Hour), Platforms: both}
	return post, teasers.Receipt{ID: "post-" + string(post.Teaser.Battle)}, false, nil
}

func battle(id teasers.BattleID, in time.Duration) teasers.Teaser {
	return teasers.Teaser{Battle: id, StartsAt: now.Add(in), Cards: []teasers.Card{{Headline: "Gain the most land"}}}
}

func media(name string) teasers.Media {
	return teasers.Media{Photos: []teasers.File{teasers.File(name + "/card-1.jpg")}, Video: teasers.File(name + "/teaser.mp4")}
}

func TestEachRenderedBattleStillDueIsSentWithItsMedia(t *testing.T) {
	ring, finale := battle("entry-7", 32*time.Hour), battle("season-1/finale", 35*time.Hour)
	sender := &fakeSender{}

	report, err := post_teasers_usecase.New(
		fakeBackend{listed: []teasers.Teaser{ring, finale}},
		fakeBackend{},
		renders{{Battle: "season-1/finale", Media: media("finale")}, {Battle: "entry-7", Media: media("ring")}},
		sender, cptime.NewFixedClock(now),
	).Execute(t.Context())

	require.NoError(t, err)
	assert.Equal(t, []teasers.Post{{Teaser: ring, Media: media("ring")}, {Teaser: finale, Media: media("finale")}}, sender.sent)
	require.Len(t, report, 2)
	assert.Equal(t, teasers.Outcome{
		Battle: "entry-7", Title: ring.Words().Title, Result: teasers.Scheduled, At: ring.StartsAt.Add(-24 * time.Hour),
		Platforms: both, Post: "post-entry-7", Refused: map[teasers.Platform]string(nil),
	}, report[0])
	assert.Equal(t, teasers.Scheduled, report[1].Result)
	assert.False(t, report.Failed())
}

func TestABattleTeasedOrDroppedSinceItsRenderIsSkippedAndOneNotRenderedToo(t *testing.T) {
	teased, dropped, late := battle("entry-6", 30*time.Hour), battle("entry-5", 31*time.Hour), battle("entry-8", 33*time.Hour)
	sender := &fakeSender{}

	report, err := post_teasers_usecase.New(
		fakeBackend{listed: []teasers.Teaser{teased, late}},
		fakeBackend{teased: []teasers.BattleID{"entry-6"}},
		renders{{Battle: teased.Battle, Media: media("6")}, {Battle: dropped.Battle, Media: media("5")}},
		sender, cptime.NewFixedClock(now),
	).Execute(t.Context())

	require.NoError(t, err)
	assert.Empty(t, sender.sent)
	assert.Equal(t, []teasers.BattleID{"entry-8", "entry-6", "entry-5"}, []teasers.BattleID{report[0].Battle, report[1].Battle, report[2].Battle})
	for _, outcome := range report {
		assert.Equal(t, teasers.Skipped, outcome.Result)
	}
	assert.Contains(t, report[0].Why, "not rendered")
	assert.Contains(t, report[1].Why, "no longer due")
	assert.False(t, report.Failed())
}

func TestAHeldTeaserIsNoFailureAndAFailedOneIs(t *testing.T) {
	held, failing := battle("entry-7", 32*time.Hour), battle("entry-8", 33*time.Hour)

	report, err := post_teasers_usecase.New(
		fakeBackend{listed: []teasers.Teaser{held, failing}}, fakeBackend{},
		renders{{Battle: held.Battle, Media: media("7")}, {Battle: failing.Battle, Media: media("8")}},
		&fakeSender{held: []teasers.BattleID{"entry-7"}, failing: []teasers.BattleID{"entry-8"}}, cptime.NewFixedClock(now),
	).Execute(t.Context())

	require.NoError(t, err)
	assert.Equal(t, []teasers.Result{teasers.Held, teasers.Failed}, []teasers.Result{report[0].Result, report[1].Result})
	assert.Equal(t, "the caption is refused", report[1].Why)
	assert.True(t, report.Failed())
}
