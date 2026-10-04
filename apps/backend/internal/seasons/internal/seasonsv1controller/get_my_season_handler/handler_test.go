package get_my_season_handler_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	seasonsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/seasons/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/seasonsv1controller/caller"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/seasonsv1controller/get_my_season_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/seasonsv1controller/get_my_season_handler/my_season_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/standings"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
)

const adaID = "0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11"

type asked struct {
	account standings.AccountID
	country string
}

type stubQuery struct {
	mine  *seasonsv1.GetMySeasonResponse
	err   error
	asked *asked
}

func (s stubQuery) MySeason(_ context.Context, account standings.AccountID, country string) (*seasonsv1.GetMySeasonResponse, error) {
	*s.asked = asked{account: account, country: country}
	return s.mine, s.err
}

func TestTheCallerReadsWhatTheQueryAnswersForItsAccountAndTheCountryAsked(t *testing.T) {
	mine := &seasonsv1.GetMySeasonResponse{CountryId: "bg", Tiles: 74, GlobalRank: 2, CountryRank: 1, CountryTiles: 3}
	query := stubQuery{mine: mine, asked: new(asked)}

	res, err := get_my_season_handler.New(query).
		GetMySeason(cpctx.AddAccountToContext(t.Context(), adaID), connect.NewRequest(&seasonsv1.GetMySeasonRequest{CountryId: "fr"}))

	require.NoError(t, err)
	assert.True(t, proto.Equal(mine, res.Msg))
	assert.Equal(t, adaID, query.asked.account.String())
	assert.Equal(t, "fr", query.asked.country)
}

func TestACallWithNoAccountIsUnauthenticatedAndReadsNothing(t *testing.T) {
	query := stubQuery{asked: new(asked)}

	_, err := get_my_season_handler.New(query).
		GetMySeason(t.Context(), connect.NewRequest(&seasonsv1.GetMySeasonRequest{}))

	assert.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
	assert.ErrorIs(t, err, caller.ErrNoAccount)
	assert.Equal(t, asked{}, *query.asked)
}

func TestACountryThatIsNotOneIsInvalidArgument(t *testing.T) {
	_, err := get_my_season_handler.New(stubQuery{
		err: fmt.Errorf("%w: %q", my_season_query.ErrUnknownCountry, "zz"), asked: new(asked),
	}).GetMySeason(cpctx.AddAccountToContext(t.Context(), adaID), connect.NewRequest(&seasonsv1.GetMySeasonRequest{CountryId: "zz"}))

	assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

func TestAFailureIsLeftToTheErrorNet(t *testing.T) {
	refused := errors.New("the database is down")

	_, err := get_my_season_handler.New(stubQuery{err: refused, asked: new(asked)}).
		GetMySeason(cpctx.AddAccountToContext(t.Context(), adaID), connect.NewRequest(&seasonsv1.GetMySeasonRequest{}))

	require.ErrorIs(t, err, refused)
}
