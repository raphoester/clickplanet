package get_hall_of_fame_handler_test

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	scoringv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/scoring/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/scoringv1controller/get_hall_of_fame_handler"
)

type hall struct {
	hall *scoringv1.GetHallOfFameResponse
	err  error
}

func (h hall) HallOfFame(context.Context) (*scoringv1.GetHallOfFameResponse, error) {
	return h.hall, h.err
}

func TestTheHallOfFameIsWhatTheQueryReads(t *testing.T) {
	read := &scoringv1.GetHallOfFameResponse{Seasons: []*scoringv1.EndedSeason{{
		Number: 0, EndedAtUnixMs: 1793487600000,
		Scores:    []*scoringv1.Score{{Rank: 1, CountryId: "fr", Points: 300, RoundsWon: 9}},
		Standings: []*scoringv1.Standing{{Rank: 1, AccountId: "0193b3a4-6f1e-7c2a-9d3e-2b4c5d6e7f80", Name: "Ada", CountryId: "fr", Tiles: 4000}},
	}}}

	res, err := get_hall_of_fame_handler.New(hall{hall: read}).
		GetHallOfFame(t.Context(), connect.NewRequest(&scoringv1.GetHallOfFameRequest{}))
	require.NoError(t, err)

	assert.True(t, proto.Equal(read, res.Msg), res.Msg)
}

func TestAHallOfFameThatCannotBeReadIsLeftToTheErrorNet(t *testing.T) {
	failed := errors.New("postgres is down")

	_, err := get_hall_of_fame_handler.New(hall{err: failed}).
		GetHallOfFame(t.Context(), connect.NewRequest(&scoringv1.GetHallOfFameRequest{}))

	require.ErrorIs(t, err, failed)
	assert.Equal(t, connect.CodeUnknown, connect.CodeOf(err))
}
