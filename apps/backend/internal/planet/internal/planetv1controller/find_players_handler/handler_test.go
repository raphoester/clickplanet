package find_players_handler_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger/usecases/find_players_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/find_players_handler"
)

type stubUseCase struct {
	in  find_players_usecase.In
	out find_players_usecase.Out
	err error
}

func (s *stubUseCase) Execute(_ context.Context, in find_players_usecase.In) (find_players_usecase.Out, error) {
	s.in = in
	return s.out, s.err
}

func find(t *testing.T, useCase *stubUseCase) (*planetv1.FindPlayersResponse, error) {
	t.Helper()

	res, err := find_players_handler.New(useCase).FindPlayers(t.Context(),
		connect.NewRequest(&planetv1.FindPlayersRequest{FlagCountryId: "ps", AreaCountryId: "il", Limit: 5}))
	if err != nil {
		return nil, fmt.Errorf("find refused: %w", err)
	}

	return res.Msg, nil
}

func TestTheRequestReachesTheUseCaseAndThePlayersComeBack(t *testing.T) {
	at := time.Date(2026, 9, 14, 12, 0, 0, 0, time.UTC)
	useCase := &stubUseCase{out: find_players_usecase.Out{Total: 7, Players: []find_players_usecase.Player{
		{Scope: "9.9.9.9", Tiles: 40, FirstAt: at, LastAt: at.Add(time.Minute), Banned: true, BannedUntil: at.Add(time.Hour), Offence: 2},
		{Scope: "2001:db8::/64", Tiles: 1, FirstAt: at, LastAt: at},
	}}}

	res, err := find(t, useCase)
	require.NoError(t, err)

	assert.Equal(t, find_players_usecase.In{Flag: "ps", Area: "il", Limit: 5}, useCase.in)
	assert.Equal(t, uint32(7), res.GetTotal())
	require.Len(t, res.GetPlayers(), 2)

	bot := res.GetPlayers()[0]
	assert.Equal(t, "9.9.9.9", bot.GetScope())
	assert.Equal(t, uint32(40), bot.GetTiles())
	assert.Equal(t, at, bot.GetFirstAt().AsTime())
	assert.Equal(t, at.Add(time.Minute), bot.GetLastAt().AsTime())
	assert.True(t, bot.GetBanned())
	assert.Equal(t, at.Add(time.Hour), bot.GetBannedUntil().AsTime())
	assert.Equal(t, uint32(2), bot.GetOffence())

	assert.Nil(t, res.GetPlayers()[1].GetBannedUntil(), "an unbanned scope has no end date to print")
}

func TestAnUnknownCountryIsInvalidArgument(t *testing.T) {
	_, err := find(t, &stubUseCase{err: fmt.Errorf("%w: %q", clicks.ErrUnknownCountry, "xx")})
	assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

func TestAnythingElseIsLeftToTheErrorNet(t *testing.T) {
	cause := errors.New("boom")

	_, err := find(t, &stubUseCase{err: cause})

	require.ErrorIs(t, err, cause)
	assert.Equal(t, connect.CodeUnknown, connect.CodeOf(err))
}
