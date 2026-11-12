package get_shown_names_handler_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	clipsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/clips/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/casts"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/clipsv1controller/get_shown_names_handler"
)

const (
	adaID = "0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11"
	bobID = "5e0c1b2a-3d4e-4f60-8a71-9b2c3d4e5f60"
)

type stubQuery struct {
	answer *clipsv1.GetShownNamesResponse
	err    error
	asked  [][]casts.AccountID
}

func (s *stubQuery) ShownNames(_ context.Context, accounts []casts.AccountID) (*clipsv1.GetShownNamesResponse, error) {
	s.asked = append(s.asked, accounts)
	return s.answer, s.err
}

func getShownNames(t *testing.T, query *stubQuery, ids ...string) (*connect.Response[clipsv1.GetShownNamesResponse], error) {
	t.Helper()

	return get_shown_names_handler.New(query).GetShownNames(t.Context(), connect.NewRequest(&clipsv1.GetShownNamesRequest{AccountIds: ids})) //nolint:wrapcheck // the test reads the handler's own error.
}

func account(t *testing.T, id string) casts.AccountID {
	t.Helper()
	account, err := casts.AccountIDOf(id)
	require.NoError(t, err)
	return account
}

func TestTheAnswerIsTheQuerysForTheAccountsInTheOrderAsked(t *testing.T) {
	answer := &clipsv1.GetShownNamesResponse{ShownNames: []*clipsv1.ShownName{{AccountId: adaID, Name: "Ada"}}}
	query := &stubQuery{answer: answer}

	res, err := getShownNames(t, query, bobID, adaID)

	require.NoError(t, err)
	assert.True(t, proto.Equal(answer, res.Msg), res.Msg.String())
	assert.Equal(t, [][]casts.AccountID{{account(t, bobID), account(t, adaID)}}, query.asked)
}

func TestAPageOfAHundredIsReadAndMoreIsRefused(t *testing.T) {
	page := strings.Split(strings.Repeat(adaID+",", get_shown_names_handler.MaxAsked), ",")[:get_shown_names_handler.MaxAsked]

	query := &stubQuery{answer: &clipsv1.GetShownNamesResponse{}}
	_, err := getShownNames(t, query, page...)
	require.NoError(t, err)
	assert.Len(t, query.asked, 1)

	query = &stubQuery{answer: &clipsv1.GetShownNamesResponse{}}
	_, err = getShownNames(t, query, append(page, bobID)...)
	assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
	assert.ErrorIs(t, err, get_shown_names_handler.ErrTooMany)
	assert.Empty(t, query.asked)
}

func TestAnIdThatIsNotAnAccountIsRefusedAndReadsNothing(t *testing.T) {
	for _, id := range []string{"", "Ada", "00000000-0000-0000-0000-000000000000"} {
		query := &stubQuery{answer: &clipsv1.GetShownNamesResponse{}}

		_, err := getShownNames(t, query, adaID, id)

		assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err), id)
		assert.Empty(t, query.asked, id)
	}
}

func TestTheAnswerIsNeverStored(t *testing.T) {
	res, err := getShownNames(t, &stubQuery{answer: &clipsv1.GetShownNamesResponse{}}, adaID)

	require.NoError(t, err)
	assert.Equal(t, "no-store", res.Header().Get("Cache-Control"), "a rename, an opt-out or a verdict holds at the next read")
}

func TestAFailedReadIsTheErrorNets(t *testing.T) {
	failure := errors.New("postgres is down")

	_, err := getShownNames(t, &stubQuery{err: failure}, adaID)

	assert.ErrorIs(t, err, failure)
}
