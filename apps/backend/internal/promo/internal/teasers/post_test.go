package teasers_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/teasers"
)

func TestAPostsKeyIsItsBattleAtItsStart(t *testing.T) {
	post := teasers.Post{Teaser: ringWar()}
	again := teasers.Post{Teaser: ringWar(), Slot: teasers.Slot{At: start.Add(-time.Hour)}}
	moved := teasers.Post{Teaser: ringWar()}
	moved.Teaser.StartsAt = start.Add(time.Hour)

	assert.Len(t, post.Key(), 64)
	assert.Equal(t, post.Key(), again.Key(), "a slot planned again is the same post")
	assert.NotEqual(t, post.Key(), moved.Key())
}

func TestAPostKeepsTheBattleItTeases(t *testing.T) {
	assert.Equal(t, map[string]any{"teaserBattle": "entry-7", "teaserStartsAt": "2026-11-14T20:00:00Z"},
		teasers.Post{Teaser: ringWar()}.Metadata())
}

func TestAReportFailsOnAFailureOrAPostEveryPlatformRefused(t *testing.T) {
	at := start.Add(-30 * time.Hour)
	scheduled := teasers.Post{Teaser: ringWar(), Slot: teasers.Slot{At: at, Platforms: both}}

	report := teasers.Report{}.
		With(scheduled, teasers.Receipt{ID: "post-1", Refused: map[teasers.Platform]string{teasers.TikTok: "no account"}}, false, nil).
		With(teasers.Post{Teaser: ringWar()}, teasers.Receipt{}, true, nil).
		WithSkipped("entry-9", "BATTLE", "not rendered")
	assert.False(t, report.Failed())
	assert.Equal(t, []teasers.Result{teasers.Scheduled, teasers.Held, teasers.Skipped},
		[]teasers.Result{report[0].Result, report[1].Result, report[2].Result})
	assert.Equal(t, "BATTLE ON SATURDAY 14 NOVEMBER AT 20:00 UTC: scheduled at 2026-11-13T14:00:00Z on [tiktok youtube]", report[0].String())

	assert.True(t, report.With(scheduled, teasers.Receipt{}, false, errors.New("refused")).Failed())
	everywhere := map[teasers.Platform]string{teasers.TikTok: "no account", teasers.YouTube: "no account"}
	assert.True(t, teasers.Report{}.With(scheduled, teasers.Receipt{Refused: everywhere}, false, nil).Failed())
}
