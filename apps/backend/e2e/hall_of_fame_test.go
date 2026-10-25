package e2e_test

import (
	"net/http"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	seasonsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/seasons/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/seasons/v1/seasonsv1connect"
)

func TestWhileTheFirstSeasonRunsTheHallOfFameIsEmpty(t *testing.T) {
	game := startGame(t)
	game.namedPlayer(t, "google-ada", "Ada").click(1, "fr")

	res, err := seasonsv1connect.NewSeasonServiceClient(http.DefaultClient, game.baseURL, connect.WithHTTPGet()).
		GetHallOfFame(t.Context(), connect.NewRequest(&seasonsv1.GetHallOfFameRequest{}))

	require.NoError(t, err)
	assert.Equal(t, "public, max-age=60", res.Header().Get("Cache-Control"))
	assert.Empty(t, res.Msg.GetSeasons())
}
