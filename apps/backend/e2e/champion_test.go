package e2e_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	chatv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/chat/v1"
	seasonsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/seasons/v1"
)

func (p *gamer) champions() []*chatv1.Announcement {
	p.t.Helper()

	var champions []*chatv1.Announcement
	for _, announcement := range p.announcements() {
		if announcement.GetKind() == "champion" {
			champions = append(champions, announcement)
		}
	}
	return champions
}

func TestTheChatAnnouncesTheCountryThatWonTheSeasonOnce(t *testing.T) {
	game := startGameWith(t, func(configs stackConfigs) {
		configs.seasons.Calendar.List[0].EndsAt = time.Now().Add(8 * time.Second)
		configs.seasons.Calendar.List[0].Finale = 6 * time.Second
	})
	player := game.newPlayer(t)

	player.click(1, "fr")

	require.Eventually(t, func() bool { return len(player.champions()) > 0 }, 20*time.Second, 100*time.Millisecond,
		"the finale closes on the first snapshot after the season ends")
	assert.JSONEq(t, `{"season":0,"countries":["fr"]}`, player.champions()[0].GetPayload())

	race := game.raceWhere(t, func(race *seasonsv1.Race) bool { return race.GetClosed().GetFinale() })
	require.NotEmpty(t, race.GetClosed().GetAfter())
	assert.Equal(t, "fr", race.GetClosed().GetAfter()[0].GetCountryId(), "the race crowns who the chat does")

	assert.Never(t, func() bool { return len(player.champions()) > 1 }, time.Second, 100*time.Millisecond,
		"the snapshots after the end tell nothing more")
}
