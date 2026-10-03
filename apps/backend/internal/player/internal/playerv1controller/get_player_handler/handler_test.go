package get_player_handler_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/usecases/get_player_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/get_player_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/titles"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/wearing"
)

type stubUseCase struct {
	player get_player_usecase.Player
	err    error
	asked  []string
}

func (s *stubUseCase) Execute(_ context.Context, name string) (get_player_usecase.Player, error) {
	s.asked = append(s.asked, name)
	return s.player, s.err
}

func getPlayer(t *testing.T, useCase *stubUseCase, name string) (*connect.Response[playerv1.GetPlayerResponse], error) {
	t.Helper()

	return get_player_handler.New(useCase).GetPlayer(t.Context(), connect.NewRequest(&playerv1.GetPlayerRequest{Name: name})) //nolint:wrapcheck // the tests read the connect error.
}

func TestThePlayerIsMappedAndMayBeCached(t *testing.T) {
	createdAt := time.Date(2026, 9, 1, 8, 30, 0, 0, time.UTC)
	useCase := &stubUseCase{player: get_player_usecase.Player{
		Player: players.Player{
			Name: "Ada_L",
			Stats: players.Stats{
				Account: players.AccountID{15: 1}, TilesTaken: 42, StreakCurrent: 3, StreakBest: 5,
				StreakLastDay: players.DayOf(time.Date(2026, 9, 14, 0, 0, 0, 0, time.UTC)),
			},
			CreatedAt: createdAt,
		},
		Titles: wearing.Showcase{
			Worn: titles.Standing{Title: titles.Warlord{}, Place: titles.Place{Track: "conquest", TrackName: "Conquest", Number: 3, Count: 5}},
			Shown: []titles.Standing{
				{Title: titles.OG{}},
				{Title: titles.Warlord{}, Place: titles.Place{Track: "conquest", TrackName: "Conquest", Number: 3, Count: 5}},
			},
		},
	}}

	res, err := getPlayer(t, useCase, "ada_l")

	require.NoError(t, err)
	assert.Equal(t, []string{"ada_l"}, useCase.asked)
	player := res.Msg.GetPlayer()
	assert.Equal(t, "Ada_L", player.GetName())
	assert.Equal(t, uint64(42), player.GetStats().GetTilesTaken())
	assert.Equal(t, uint32(3), player.GetStats().GetStreakCurrent())
	assert.Equal(t, uint32(5), player.GetStats().GetStreakBest())
	assert.Equal(t, "2026-09-14", player.GetStats().GetStreakLastDay())
	assert.Equal(t, createdAt.UnixMilli(), player.GetCreatedAtUnixMs())
	require.Len(t, player.GetTitles(), 2)
	assert.Equal(t, "og", player.GetTitles()[0].GetId())
	assert.Nil(t, player.GetTitles()[0].GetRank(), "a title outside any track has no rank")
	assert.Equal(t, "Warlord", player.GetWornTitle().GetName())
	assert.Equal(t, &playerv1.Rank{TrackId: "conquest", TrackName: "Conquest", Number: 3, Count: 5}, player.GetWornTitle().GetRank())
	assert.Equal(t, "public, max-age=10", res.Header().Get("Cache-Control"))
}

func TestAnUnknownCreationDateIsZero(t *testing.T) {
	res, err := getPlayer(t, &stubUseCase{player: get_player_usecase.Player{Player: players.Player{Name: "Ada_L"}}}, "Ada_L")

	require.NoError(t, err)
	assert.Zero(t, res.Msg.GetPlayer().GetCreatedAtUnixMs())
}

func TestANameNobodyHoldsIsNotFound(t *testing.T) {
	_, err := getPlayer(t, &stubUseCase{err: players.ErrNoProfile}, "Bob")

	assert.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
}

func TestAStoreFailureIsNotNotFound(t *testing.T) {
	_, err := getPlayer(t, &stubUseCase{err: errors.New("postgres is down")}, "Ada_L")

	require.Error(t, err)
	assert.NotEqual(t, connect.CodeNotFound, connect.CodeOf(err))
}
