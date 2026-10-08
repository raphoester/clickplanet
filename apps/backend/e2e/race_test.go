package e2e_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	seasonsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/seasons/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/seasons/v1/seasonsv1connect"
)

func (s gameStack) raceWhere(t *testing.T, done func(*seasonsv1.Race) bool) *seasonsv1.Race {
	t.Helper()

	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	stream, err := seasonsv1connect.NewSeasonServiceClient(http.DefaultClient, s.baseURL).
		ListenForEvents(ctx, connect.NewRequest(&seasonsv1.ListenForEventsRequest{}))
	require.NoError(t, err)
	defer func() { _ = stream.Close() }()

	for stream.Receive() {
		if race := stream.Msg().GetRace(); race != nil && done(race) {
			return race
		}
	}
	require.FailNow(t, "the stream ended before the race it waited for", "%v", stream.Err())
	return nil
}

func TestTheRaceRanksTheCountriesByTheGroundTheyHoldToday(t *testing.T) {
	game := startGame(t)
	player := game.newPlayer(t)

	player.click(1, "fr")
	player.click(2, "fr")
	player.click(3, "de")

	race := game.raceWhere(t, func(race *seasonsv1.Race) bool {
		return len(race.GetRound().GetStandings()) == 2
	})

	round := race.GetRound()
	assert.Equal(t, uint32(1), round.GetNumber())
	assert.Greater(t, round.GetEndsAtUnixMs(), time.Now().UnixMilli())
	assert.False(t, round.GetFinale())

	standings := round.GetStandings()
	assert.Equal(t, uint32(1), standings[0].GetRank())
	assert.Equal(t, "fr", standings[0].GetCountryId())
	assert.Equal(t, uint32(25), standings[0].GetPoints())
	assert.Equal(t, uint32(2), standings[1].GetRank())
	assert.Equal(t, "de", standings[1].GetCountryId())
	assert.Equal(t, uint32(18), standings[1].GetPoints())

	assert.LessOrEqual(t, standings[0].GetShare(), 2.0/mapTiles, "an average over every snapshot, the ones before the clicks too")
	assert.LessOrEqual(t, standings[1].GetShare(), 1.0/mapTiles)
	assert.Greater(t, standings[1].GetShare(), 0.0)

	assert.Empty(t, race.GetScores(), "no round has closed yet")
}
