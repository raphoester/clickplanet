package get_round_players_handler_test

import (
	"context"
	"fmt"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	scoringv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/scoring/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/books"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/scoringv1controller/get_round_players_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/scoringv1controller/get_round_players_handler/round_players_query"
)

type stubQuery struct {
	players []*scoringv1.RoundPlayer
	err     error
}

func (s stubQuery) Players(context.Context, books.RoundID) ([]*scoringv1.RoundPlayer, error) {
	return s.players, s.err
}

func TestThePlayersOfTheRoundAskedAreAnswered(t *testing.T) {
	res, err := get_round_players_handler.New(stubQuery{players: []*scoringv1.RoundPlayer{{CountryId: "fr", Tiles: 3}}}).
		GetRoundPlayers(t.Context(), connect.NewRequest(&scoringv1.GetRoundPlayersRequest{RoundId: "season-0/finale"}))

	require.NoError(t, err)
	require.Len(t, res.Msg.GetPlayers(), 1)
	assert.Equal(t, "fr", res.Msg.GetPlayers()[0].GetCountryId())
}

func TestARoundNeverOpenedIsNotFoundAndNoIDIsInvalid(t *testing.T) {
	_, err := get_round_players_handler.New(stubQuery{err: fmt.Errorf("%w: %q", round_players_query.ErrNoRound, "season-9/finale")}).
		GetRoundPlayers(t.Context(), connect.NewRequest(&scoringv1.GetRoundPlayersRequest{RoundId: "season-9/finale"}))
	assert.Equal(t, connect.CodeNotFound, connect.CodeOf(err))

	_, err = get_round_players_handler.New(stubQuery{}).
		GetRoundPlayers(t.Context(), connect.NewRequest(&scoringv1.GetRoundPlayersRequest{}))
	assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}
