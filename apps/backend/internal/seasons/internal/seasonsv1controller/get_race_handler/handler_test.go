package get_race_handler_test

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	seasonsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/seasons/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/seasonsv1controller/get_race_handler"
)

type race struct {
	race *seasonsv1.Race
	err  error
}

func (r race) Race(context.Context) (*seasonsv1.Race, error) {
	return r.race, r.err
}

func TestTheRaceIsTheOneTheStreamSends(t *testing.T) {
	read := &seasonsv1.Race{
		Round: &seasonsv1.Round{Number: 6, EndsAtUnixMs: 1792357200000},
		Closed: &seasonsv1.ClosedRound{
			Season: 1, Number: 5, EndedAtUnixMs: 1792270800000,
			Standings: []*seasonsv1.RoundStanding{{Rank: 1, CountryId: "fr", Share: 0.18, Points: 25}},
		},
	}

	res, err := get_race_handler.New(race{race: read}).
		GetRace(t.Context(), connect.NewRequest(&seasonsv1.GetRaceRequest{}))
	require.NoError(t, err)

	assert.True(t, proto.Equal(read, res.Msg.GetRace()))
}

func TestNoSeasonAndNoRoundClosedIsAnEmptyRace(t *testing.T) {
	res, err := get_race_handler.New(race{race: &seasonsv1.Race{}}).
		GetRace(t.Context(), connect.NewRequest(&seasonsv1.GetRaceRequest{}))
	require.NoError(t, err)

	assert.Nil(t, res.Msg.GetRace().GetClosed())
	assert.Nil(t, res.Msg.GetRace().GetRound())
}

func TestARaceThatCannotBeReadIsAnError(t *testing.T) {
	failed := errors.New("postgres is down")

	_, err := get_race_handler.New(race{err: failed}).
		GetRace(t.Context(), connect.NewRequest(&seasonsv1.GetRaceRequest{}))

	require.ErrorIs(t, err, failed)
}
