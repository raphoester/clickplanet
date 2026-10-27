package get_standings_handler_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	scoringv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/scoring/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/scoringv1controller/get_standings_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/scoringv1controller/get_standings_handler/standings_query"
)

type stubQuery struct {
	standings *scoringv1.GetStandingsResponse
	err       error
	country   *string
}

func (s stubQuery) Standings(_ context.Context, country string) (*scoringv1.GetStandingsResponse, error) {
	*s.country = country
	return s.standings, s.err
}

func TestGetStandingsAnswersWhatTheQueryReadsForTheCountryAsked(t *testing.T) {
	standings := &scoringv1.GetStandingsResponse{Standings: []*scoringv1.Standing{{Rank: 1, Name: "Ada", CountryId: "fr", Tiles: 3}}}
	country := new(string)

	res, err := get_standings_handler.New(stubQuery{standings: standings, country: country}).
		GetStandings(t.Context(), connect.NewRequest(&scoringv1.GetStandingsRequest{CountryId: "fr"}))

	require.NoError(t, err)
	assert.True(t, proto.Equal(standings, res.Msg))
	assert.Equal(t, "fr", *country)
}

func TestACountryThatIsNotOneIsInvalidArgument(t *testing.T) {
	_, err := get_standings_handler.New(stubQuery{
		err: fmt.Errorf("%w: %q", standings_query.ErrUnknownCountry, "zz"), country: new(string),
	}).GetStandings(t.Context(), connect.NewRequest(&scoringv1.GetStandingsRequest{CountryId: "zz"}))

	assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

func TestAFailureIsLeftToTheErrorNet(t *testing.T) {
	refused := errors.New("the database is down")

	_, err := get_standings_handler.New(stubQuery{err: refused, country: new(string)}).
		GetStandings(t.Context(), connect.NewRequest(&scoringv1.GetStandingsRequest{}))

	require.ErrorIs(t, err, refused)
	assert.Equal(t, connect.CodeUnknown, connect.CodeOf(err))
}
