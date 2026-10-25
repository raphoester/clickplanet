package e2e_test

import (
	"net/http"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1/planetv1connect"
	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
)

func (p *gamer) track(id string) *playerv1.Track {
	p.t.Helper()

	req := connect.NewRequest(&playerv1.GetTitlesRequest{})
	p.send(req.Header())
	res, err := p.players().GetTitles(p.t.Context(), req)
	require.NoError(p.t, err)
	for _, track := range res.Msg.GetTracks() {
		if track.GetId() == id {
			return track
		}
	}
	return nil
}

func (p *gamer) fortifyAnIslandAndDropABomb() {
	p.t.Helper()

	members, _ := landmassesOf(p.t)
	var island []uint32
	for id, tiles := range members {
		if id != 0 && len(tiles) == 3 {
			island = tiles
			break
		}
	}
	require.Len(p.t, island, 3, "the map has a landmass of three tiles")
	for _, tile := range island {
		p.click(tile, "pt")
	}

	grant := connect.NewRequest(&planetv1.GrantChargesRequest{AccountId: p.account(), Bomb: true})
	_, err := planetv1connect.NewAdminServiceClient(http.DefaultClient, p.stack.consoleURL).GrantCharges(p.t.Context(), grant)
	require.NoError(p.t, err)
	drop := connect.NewRequest(&planetv1.DropBombRequest{Target: pointOf(p.t, 1), CountryId: "pt"})
	p.send(drop.Header())
	_, err = planetv1connect.NewClickServiceClient(http.DefaultClient, p.stack.baseURL).DropBomb(p.t.Context(), drop)
	require.NoError(p.t, err)
}

func (p *gamer) climbedBothTracks() bool {
	bombs := p.track("bombs")
	return p.track("defense").GetProgress() == 3 && bombs.GetProgress() == 1 && bombs.GetSteps()[0].GetEarned()
}

func TestAFortifiedLandmassAndADroppedBombClimbTheirTracks(t *testing.T) {
	game := startGame(t)
	ada := game.newPlayer(t)
	ada.link("google-ada")

	ada.fortifyAnIslandAndDropABomb()

	require.Eventually(t, ada.climbedBothTracks, 5*time.Second, 20*time.Millisecond,
		"planet tells each fortify and each bomb with its account, and player counts them")
}
