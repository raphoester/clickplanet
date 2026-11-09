package teasers_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/teasers"
)

// One photo a card of ringWar's, and a video.
func media() teasers.Media {
	return teasers.Media{
		Photos: []teasers.MediaURL{"https://media.example/card-1.jpg", "https://media.example/card-2.jpg"},
		Video:  "https://media.example/teaser.mp4",
	}
}

func post(t *testing.T, teaser teasers.Teaser) teasers.Post {
	t.Helper()
	made, err := teasers.NewPost(teaser, media())
	require.NoError(t, err)
	return made
}

func TestAPostIsOnePhotoACardAndAVideo(t *testing.T) {
	_, err := teasers.NewPost(ringWar(), teasers.Media{Photos: media().Photos[:1]})

	require.ErrorIs(t, err, teasers.ErrInvalidPost)
	assert.ErrorContains(t, err, "no video")
	assert.ErrorContains(t, err, "1 photos for 2 cards")
}

func TestAPostsKeyIsItsBattleAtItsStart(t *testing.T) {
	one := post(t, ringWar())
	again := one.At(teasers.Slot{At: start.Add(-time.Hour)})
	moved := ringWar()
	moved.StartsAt = start.Add(time.Hour)

	assert.Len(t, one.Key(), 64)
	assert.Equal(t, one.Key(), again.Key(), "a slot planned again is the same post")
	assert.NotEqual(t, one.Key(), post(t, moved).Key())
}

func TestAPostKeepsTheBattleItTeases(t *testing.T) {
	assert.Equal(t, map[string]any{"teaserBattle": "entry-7", "teaserStartsAt": "2026-11-14T20:00:00Z"}, post(t, ringWar()).Metadata())
}

func TestAPostedTeaserSaysWhereWhenAndWhoRefusedIt(t *testing.T) {
	at := start.Add(-30 * time.Hour)
	refused := map[teasers.Platform]string{teasers.YouTube: "no account"}

	outcome := teasers.PostedOf(post(t, ringWar()).At(teasers.Slot{At: at, Platforms: both}), teasers.Receipt{ID: "post-1", Refused: refused})

	assert.Equal(t, teasers.Outcome{
		Result: teasers.Posted, Title: "BATTLE ON SATURDAY 14 NOVEMBER AT 20:00 UTC", At: at, Platforms: both,
		Post: "post-1", Refused: refused,
	}, outcome)
}
