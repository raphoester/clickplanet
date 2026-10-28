package get_my_tally_handler_test

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
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/books"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/scoringv1controller/caller"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/scoringv1controller/get_my_tally_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/scoringv1controller/get_my_tally_handler/my_tally_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/standings"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
)

const adaID = "0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11"

type asked struct {
	book    books.BookID
	account standings.AccountID
	country string
}

type stubQuery struct {
	mine  *scoringv1.GetMyTallyResponse
	err   error
	asked *asked
}

func (s stubQuery) MyTally(_ context.Context, book books.BookID, account standings.AccountID, country string) (*scoringv1.GetMyTallyResponse, error) {
	*s.asked = asked{book: book, account: account, country: country}
	return s.mine, s.err
}

func TestTheCallerReadsWhatTheQueryAnswersForItsAccountAndTheCountryAsked(t *testing.T) {
	mine := &scoringv1.GetMyTallyResponse{CountryId: "bg", Tiles: 74, GlobalRank: 2, CountryRank: 1, CountryTiles: 3}
	query := stubQuery{mine: mine, asked: new(asked)}

	res, err := get_my_tally_handler.New(query).
		GetMyTally(cpctx.AddAccountToContext(t.Context(), adaID), connect.NewRequest(&scoringv1.GetMyTallyRequest{BookId: "season-0", CountryId: "fr"}))

	require.NoError(t, err)
	assert.True(t, proto.Equal(mine, res.Msg))
	assert.Equal(t, adaID, query.asked.account.String())
	assert.Equal(t, "fr", query.asked.country)
	assert.Equal(t, books.BookID("season-0"), query.asked.book)
}

func TestACallWithNoAccountIsUnauthenticatedAndReadsNothing(t *testing.T) {
	query := stubQuery{asked: new(asked)}

	_, err := get_my_tally_handler.New(query).
		GetMyTally(t.Context(), connect.NewRequest(&scoringv1.GetMyTallyRequest{}))

	assert.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
	assert.ErrorIs(t, err, caller.ErrNoAccount)
	assert.Equal(t, asked{}, *query.asked)
}

func TestACountryThatIsNotOneIsInvalidArgument(t *testing.T) {
	_, err := get_my_tally_handler.New(stubQuery{
		err: fmt.Errorf("%w: %q", my_tally_query.ErrUnknownCountry, "zz"), asked: new(asked),
	}).GetMyTally(cpctx.AddAccountToContext(t.Context(), adaID), connect.NewRequest(&scoringv1.GetMyTallyRequest{BookId: "season-0", CountryId: "zz"}))

	assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

func TestAFailureIsLeftToTheErrorNet(t *testing.T) {
	refused := errors.New("the database is down")

	_, err := get_my_tally_handler.New(stubQuery{err: refused, asked: new(asked)}).
		GetMyTally(cpctx.AddAccountToContext(t.Context(), adaID), connect.NewRequest(&scoringv1.GetMyTallyRequest{BookId: "season-0"}))

	require.ErrorIs(t, err, refused)
}

func TestABookThatIsNotOneIsInvalidArgumentAndReadsNothing(t *testing.T) {
	query := stubQuery{asked: new(asked)}

	_, err := get_my_tally_handler.New(query).
		GetMyTally(cpctx.AddAccountToContext(t.Context(), adaID), connect.NewRequest(&scoringv1.GetMyTallyRequest{}))

	assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
	assert.Equal(t, asked{}, *query.asked)
}
