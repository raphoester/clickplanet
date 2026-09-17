package get_stats_handler_test

import (
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/inmemory_player_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/usecases/get_stats_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/get_stats_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

func TestTheCallersStatsAreAnswered(t *testing.T) {
	storage := inmemory_player_storage.New(inmemory_player_storage.NewMemoryPersistence())
	at := time.Date(2026, 9, 17, 10, 0, 0, 0, time.UTC)
	ada, err := players.AccountIDOf("0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11")
	require.NoError(t, err)
	storage.RecordTake(ada, at)
	storage.RecordTake(ada, at)

	res, err := get_stats_handler.New(get_stats_usecase.New(storage, cptime.NewFixedClock(at))).
		GetStats(cpctx.AddAccountToContext(t.Context(), ada.String()), connect.NewRequest(&playerv1.GetStatsRequest{}))

	require.NoError(t, err)
	assert.True(t, proto.Equal(&playerv1.Stats{TilesTaken: 2, StreakCurrent: 1, StreakBest: 1, StreakLastDay: "2026-09-17"},
		res.Msg.GetStats()))
}
