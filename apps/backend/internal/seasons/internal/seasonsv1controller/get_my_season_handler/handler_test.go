package get_my_season_handler_test

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	seasonsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/seasons/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/seasonsv1controller/caller"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/seasonsv1controller/get_my_season_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/standings"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
)

const adaID = "0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11"

type stubQuery struct {
	mine    *seasonsv1.GetMySeasonResponse
	err     error
	account *standings.AccountID
}

func (s stubQuery) MySeason(_ context.Context, account standings.AccountID) (*seasonsv1.GetMySeasonResponse, error) {
	*s.account = account
	return s.mine, s.err
}

func TestTheCallerReadsWhatTheQueryAnswersForItsAccount(t *testing.T) {
	mine := &seasonsv1.GetMySeasonResponse{CountryId: "fr", Tiles: 2, GlobalRank: 2, CountryRank: 1}
	account := new(standings.AccountID)

	res, err := get_my_season_handler.New(stubQuery{mine: mine, account: account}).
		GetMySeason(cpctx.AddAccountToContext(t.Context(), adaID), connect.NewRequest(&seasonsv1.GetMySeasonRequest{}))

	require.NoError(t, err)
	assert.True(t, proto.Equal(mine, res.Msg))
	assert.Equal(t, adaID, account.String())
}

func TestACallWithNoAccountIsUnauthenticatedAndReadsNothing(t *testing.T) {
	account := new(standings.AccountID)

	_, err := get_my_season_handler.New(stubQuery{account: account}).
		GetMySeason(t.Context(), connect.NewRequest(&seasonsv1.GetMySeasonRequest{}))

	assert.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
	assert.ErrorIs(t, err, caller.ErrNoAccount)
	assert.Equal(t, standings.AccountID{}, *account)
}

func TestAFailureIsLeftToTheErrorNet(t *testing.T) {
	refused := errors.New("the database is down")

	_, err := get_my_season_handler.New(stubQuery{err: refused, account: new(standings.AccountID)}).
		GetMySeason(cpctx.AddAccountToContext(t.Context(), adaID), connect.NewRequest(&seasonsv1.GetMySeasonRequest{}))

	require.ErrorIs(t, err, refused)
}
