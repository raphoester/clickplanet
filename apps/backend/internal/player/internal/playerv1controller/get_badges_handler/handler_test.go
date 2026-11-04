package get_badges_handler_test

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/get_badges_handler"
)

const (
	adaID = "0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11"
	bobID = "5e0c1b2a-3d4e-4f60-8a71-9b2c3d4e5f60"
)

type stubQuery struct {
	answer *playerv1.GetBadgesResponse
	err    error
	asked  [][]players.AccountID
}

func (s *stubQuery) Badges(_ context.Context, accounts []players.AccountID) (*playerv1.GetBadgesResponse, error) {
	s.asked = append(s.asked, accounts)
	return s.answer, s.err
}

func getBadges(t *testing.T, query *stubQuery, ids ...string) (*connect.Response[playerv1.GetBadgesResponse], error) {
	t.Helper()

	return get_badges_handler.New(query).GetBadges(t.Context(), connect.NewRequest(&playerv1.GetBadgesRequest{AccountIds: ids})) //nolint:wrapcheck // the test reads the handler's own error.
}

func account(t *testing.T, id string) players.AccountID {
	t.Helper()

	parsed, err := players.AccountIDOf(id)
	require.NoError(t, err)
	return parsed
}

func TestTheBadgesAreTheQuerysForTheAccountsAsked(t *testing.T) {
	query := &stubQuery{answer: &playerv1.GetBadgesResponse{Badges: []*playerv1.Badge{{AccountId: adaID, Streak: 4}}}}

	res, err := getBadges(t, query, adaID, bobID)

	require.NoError(t, err)
	assert.True(t, proto.Equal(query.answer, res.Msg))
	assert.Equal(t, [][]players.AccountID{{account(t, adaID), account(t, bobID)}}, query.asked)
}

func TestAnIdThatIsNotAnAccountIsRefusedAndReadsNothing(t *testing.T) {
	for _, id := range []string{"", "not-an-account", "00000000-0000-0000-0000-000000000000"} {
		query := &stubQuery{answer: &playerv1.GetBadgesResponse{}}

		_, err := getBadges(t, query, adaID, id)

		assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err), id)
		assert.Empty(t, query.asked, id)
	}
}

func TestAPageOfAccountsIsAnsweredAndAnyMoreIsRefused(t *testing.T) {
	page := make([]string, get_badges_handler.MaxAsked)
	for i := range page {
		page[i] = adaID
	}

	query := &stubQuery{answer: &playerv1.GetBadgesResponse{}}
	_, err := getBadges(t, query, page...)
	require.NoError(t, err)

	query = &stubQuery{answer: &playerv1.GetBadgesResponse{}}
	_, err = getBadges(t, query, append(page, bobID)...)
	assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
	assert.ErrorIs(t, err, get_badges_handler.ErrTooMany)
	assert.Empty(t, query.asked, "a call too big reads nothing")
}

func TestAskingAboutNobodyIsNoError(t *testing.T) {
	query := &stubQuery{answer: &playerv1.GetBadgesResponse{}}

	_, err := getBadges(t, query)

	assert.NoError(t, err)
}

func TestTheAnswerIsNeverStored(t *testing.T) {
	res, err := getBadges(t, &stubQuery{answer: &playerv1.GetBadgesResponse{}}, adaID)

	require.NoError(t, err)
	assert.Equal(t, "no-store", res.Header().Get("Cache-Control"), "the page keeps the badges itself")
}

func TestAFailedReadIsTheErrorNets(t *testing.T) {
	failure := errors.New("postgres is down")

	_, err := getBadges(t, &stubQuery{err: failure}, adaID)

	assert.ErrorIs(t, err, failure)
	assert.NotEqual(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}
