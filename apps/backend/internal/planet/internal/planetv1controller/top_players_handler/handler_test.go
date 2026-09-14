package top_players_handler_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger/usecases/top_players_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/top_players_handler"
)

type stubUseCase struct {
	in  top_players_usecase.In
	out top_players_usecase.Out
	err error
}

func (s *stubUseCase) Execute(_ context.Context, in top_players_usecase.In) (top_players_usecase.Out, error) {
	s.in = in
	return s.out, s.err
}

func TestTheLimitReachesTheUseCaseAndThePlayersComeBack(t *testing.T) {
	at := time.Date(2026, 9, 14, 12, 0, 0, 0, time.UTC)
	useCase := &stubUseCase{out: top_players_usecase.Out{Total: 7, Players: []ledger.Player{
		{Scope: "9.9.9.9", Tiles: 40, FirstAt: at, LastAt: at.Add(time.Minute), Banned: true, BannedUntil: at.Add(time.Hour), Offence: 2},
		{Scope: "2001:db8::/64", Tiles: 1, FirstAt: at, LastAt: at},
	}}}

	res, err := top_players_handler.New(useCase).TopPlayers(t.Context(),
		connect.NewRequest(&planetv1.TopPlayersRequest{Limit: 5}))
	require.NoError(t, err)

	assert.Equal(t, top_players_usecase.In{Limit: 5}, useCase.in)
	assert.Equal(t, uint32(7), res.Msg.GetTotal())
	require.Len(t, res.Msg.GetPlayers(), 2)

	top := res.Msg.GetPlayers()[0]
	assert.Equal(t, "9.9.9.9", top.GetScope())
	assert.Equal(t, uint32(40), top.GetTiles())
	assert.Equal(t, at, top.GetFirstAt().AsTime())
	assert.Equal(t, at.Add(time.Minute), top.GetLastAt().AsTime())
	assert.True(t, top.GetBanned())
	assert.Equal(t, at.Add(time.Hour), top.GetBannedUntil().AsTime())
	assert.Equal(t, uint32(2), top.GetOffence())
	assert.Equal(t, time.Minute, top.GetActiveFor().AsDuration())
	assert.InDelta(t, 40.0, top.GetTilesPerMinute(), 1e-9)

	assert.Nil(t, res.Msg.GetPlayers()[1].GetBannedUntil(), "an unbanned scope has no end date to print")
}

func TestAnErrorIsLeftToTheErrorNet(t *testing.T) {
	cause := errors.New("boom")

	_, err := top_players_handler.New(&stubUseCase{err: cause}).TopPlayers(t.Context(),
		connect.NewRequest(&planetv1.TopPlayersRequest{}))

	require.ErrorIs(t, err, cause)
	assert.Equal(t, connect.CodeUnknown, connect.CodeOf(err))
}
