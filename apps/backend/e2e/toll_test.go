package e2e_test

import (
	"net/http"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1/planetv1connect"
)

func (p *gamer) budget(country string) *planetv1.ClickBudget {
	p.t.Helper()

	req := connect.NewRequest(&planetv1.GetBudgetRequest{CountryId: country})
	p.send(req.Header())
	res, err := planetv1connect.NewClickServiceClient(http.DefaultClient, p.stack.baseURL).GetBudget(p.t.Context(), req)
	require.NoError(p.t, err)
	return res.Msg.GetBudget()
}

func TestAClickIsPricedByThePlayersMainFlag(t *testing.T) {
	game := startGame(t)
	ada := game.newPlayer(t)

	for tile := range uint32(5) {
		ada.click(tile+1, "fr")
	}

	require.Eventually(t, func() bool { return ada.budget("es").GetCountry() == "fr" }, 5*time.Second, 20*time.Millisecond,
		"five tiles for France make France the main flag, whatever is picked next")

	bob := game.newPlayer(t)
	assert.Equal(t, "es", bob.budget("es").GetCountry(), "a player with no takes is priced by the flag it picks")
}
