package e2e_test

import (
	"slices"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
)

func (p *gamer) titles() []string {
	p.t.Helper()

	req := connect.NewRequest(&playerv1.GetTitlesRequest{})
	p.send(req.Header())
	res, err := p.players().GetTitles(p.t.Context(), req)
	require.NoError(p.t, err)
	ids := make([]string, 0, len(res.Msg.GetWearable()))
	for _, title := range res.Msg.GetWearable() {
		ids = append(ids, title.GetId())
	}
	return ids
}

func (p *gamer) holds(ids ...string) func() bool {
	return func() bool {
		held := p.titles()
		return !slices.ContainsFunc(ids, func(id string) bool { return !slices.Contains(held, id) })
	}
}

func (s gameStack) recordedASeason(t *testing.T, player *gamer) func() bool {
	return func() bool {
		var seasons int
		err := s.schema(t, "player").QueryRowContext(t.Context(),
			`SELECT count(*) FROM seasons_played WHERE account_id = $1`, player.account()).Scan(&seasons)
		return err == nil && seasons == 1
	}
}

func TestEverySignedInPlayerOfTheSeasonIsAVeteranAndThoseOfTheChampionAreChampions(t *testing.T) {
	game := startGameWith(t, func(configs stackConfigs) {
		configs.seasons.Calendar.List[0].EndsAt = time.Now().Add(12 * time.Second)
		configs.seasons.Calendar.List[0].Finale = 10 * time.Second
	})
	ada, bob, guest := game.newPlayer(t), game.newPlayer(t), game.newPlayer(t)
	ada.link("google-ada")
	bob.link("google-bob")

	ada.click(1, "fr")
	ada.click(2, "fr")
	ada.click(3, "fr")
	bob.click(4, "de")
	guest.click(5, "fr")

	require.Eventually(t, ada.holds("veteran_s0", "champion_s0"), 25*time.Second, 100*time.Millisecond,
		"France held the most ground: its players are told so once the finale closes")
	require.Eventually(t, bob.holds("veteran_s0"), 5*time.Second, 50*time.Millisecond)
	assert.NotContains(t, bob.titles(), "champion_s0")

	require.Eventually(t, game.recordedASeason(t, guest), 5*time.Second, 50*time.Millisecond, "a guest's season is kept too")
	assert.NotContains(t, guest.titles(), "veteran_s0")
	assert.NotContains(t, guest.titles(), "champion_s0")

	guest.link("google-guest")

	require.Eventually(t, guest.holds("veteran_s0", "champion_s0"), 5*time.Second, 50*time.Millisecond,
		"signing in grants them, with no take: the map is frozen after the season")
}
