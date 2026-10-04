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
	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1/playerv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
)

func (p *gamer) account() cpsession.AccountID {
	p.t.Helper()

	_, public := cpsession.TestKeyPair()
	verifier, err := cpsession.NewVerifier(public)
	require.NoError(p.t, err)
	claims, err := verifier.Verify(p.token, callerIP, time.Now())
	require.NoError(p.t, err)
	return claims.Account
}

func (p *gamer) tiles() uint64 {
	p.t.Helper()

	return p.stats().GetTilesTaken()
}

func TestTheStatsCountEachTakeOnceAcrossARestartAndARebuildDropsTheTakesReverted(t *testing.T) {
	postgres := cppg.StartTestServer(t)
	game, stop := startGameOn(t, postgres)
	ada, bob := game.newPlayer(t), game.newPlayer(t)
	ada.click(1, "fr")
	ada.click(2, "fr")
	bob.click(3, "de")
	require.Eventually(t, func() bool { return ada.tiles() == 2 && bob.tiles() == 1 }, 5*time.Second, 20*time.Millisecond)

	stop()
	game, _ = startGameOn(t, postgres)
	ada.stack, bob.stack = game, game
	ada.click(4, "fr")
	require.Eventually(t, func() bool { return ada.tiles() == 3 }, 5*time.Second, 20*time.Millisecond,
		"a take after the restart counts on the stats kept before it")
	assert.Never(t, func() bool { return ada.tiles() != 3 || bob.tiles() != 1 }, 300*time.Millisecond, 20*time.Millisecond,
		"no take before the restart is counted twice")

	_, err := planetv1connect.NewAdminServiceClient(http.DefaultClient, game.adminURL).RevertPlayer(t.Context(),
		connect.NewRequest(&planetv1.RevertPlayerRequest{AccountId: bob.account().String()}))
	require.NoError(t, err)
	planet := game.schema(t, "planet")
	require.Eventually(t, func() bool {
		var reverted int
		err := planet.QueryRowContext(t.Context(), `SELECT count(*) FROM ledger_takes WHERE reverted`).Scan(&reverted)
		return err == nil && reverted == 1
	}, 5*time.Second, 20*time.Millisecond)
	assert.Equal(t, uint64(1), bob.tiles(), "the live count keeps what it counted")

	admin := playerv1connect.NewAdminServiceClient(http.DefaultClient, game.adminURL)
	_, err = playerv1connect.NewAdminServiceClient(http.DefaultClient, game.baseURL).
		RebuildStats(t.Context(), connect.NewRequest(&playerv1.RebuildStatsRequest{}))
	require.Equal(t, connect.CodeUnimplemented, connect.CodeOf(err), "the public router does not serve it")
	rebuilt, err := admin.RebuildStats(t.Context(), connect.NewRequest(&playerv1.RebuildStatsRequest{}))
	require.NoError(t, err)
	assert.Zero(t, rebuilt.Msg.GetFromPosition(), "a fresh game's feed starts at its first take")

	require.Eventually(t, func() bool { return bob.tiles() == 0 && ada.tiles() == 3 }, 5*time.Second, 20*time.Millisecond,
		"the rebuild counts every take again but the one reverted")
}
