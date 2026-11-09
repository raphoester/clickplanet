package teasers_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/teasers"
)

var start = time.Date(2026, 11, 14, 20, 0, 0, 0, time.UTC)

func ringWar() teasers.Teaser {
	return teasers.Teaser{
		Battle: "entry-7", StartsAt: start, EndsAt: start.Add(time.Hour), Description: "Close a ring around any land.",
		Times: 3, Cards: []teasers.Card{
			{BackgroundURL: "https://media.test/ring.jpg", Headline: "Circle any land with your flag", Line: "Every click can close a ring."},
			{BackgroundURL: "https://media.test/points.jpg", Headline: "Gain the most land"},
		},
	}
}

func TestATeaserGoesOutFromADayAndAHalfToAnHourBeforeItsBattle(t *testing.T) {
	notBefore, by := ringWar().Window()

	assert.Equal(t, time.Date(2026, 11, 13, 14, 0, 0, 0, time.UTC), notBefore)
	assert.Equal(t, time.Date(2026, 11, 14, 19, 0, 0, 0, time.UTC), by)
}

func TestThePostsReadForWhatWasTeasedReachTwoLeadsBack(t *testing.T) {
	now := time.Date(2026, 11, 13, 7, 43, 0, 0, time.UTC)

	assert.Equal(t, now.Add(-60*time.Hour), teasers.TeasedSince(now))
}
