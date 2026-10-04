package get_authors_handler_test

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
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/get_authors_handler"
)

const (
	adaID   = "0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11"
	guestID = "5e0c1b2a-3d4e-4f60-8a71-9b2c3d4e5f60"
)

type stubQuery struct {
	answer *playerv1.GetAuthorsResponse
	err    error
	asked  [][]players.AccountID
}

func (s *stubQuery) Authors(_ context.Context, accounts []players.AccountID) (*playerv1.GetAuthorsResponse, error) {
	s.asked = append(s.asked, accounts)
	return s.answer, s.err
}

func getAuthors(t *testing.T, query *stubQuery, ids ...string) (*connect.Response[playerv1.GetAuthorsResponse], error) {
	t.Helper()

	return get_authors_handler.New(query).GetAuthors(t.Context(), connect.NewRequest(&playerv1.GetAuthorsRequest{AccountIds: ids})) //nolint:wrapcheck // the test reads the handler's own error.
}

func TestTheAuthorsAreTheQuerysForTheAccountsAsked(t *testing.T) {
	query := &stubQuery{answer: &playerv1.GetAuthorsResponse{Authors: []*playerv1.Author{{AccountId: adaID, Name: "Ada_L"}}}}

	res, err := getAuthors(t, query, adaID, guestID)

	require.NoError(t, err)
	assert.True(t, proto.Equal(query.answer, res.Msg))
	ada, err := players.AccountIDOf(adaID)
	require.NoError(t, err)
	guest, err := players.AccountIDOf(guestID)
	require.NoError(t, err)
	assert.Equal(t, [][]players.AccountID{{ada, guest}}, query.asked)
}

func TestAnIdThatIsNotAnAccountIsRefusedAndReadsNothing(t *testing.T) {
	for _, id := range []string{"", "not-an-account", "00000000-0000-0000-0000-000000000000"} {
		query := &stubQuery{answer: &playerv1.GetAuthorsResponse{}}

		_, err := getAuthors(t, query, adaID, id)

		assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err), id)
		assert.Empty(t, query.asked, id)
	}
}

func TestTheAnswerIsNeverStored(t *testing.T) {
	res, err := getAuthors(t, &stubQuery{answer: &playerv1.GetAuthorsResponse{}}, adaID)

	require.NoError(t, err)
	assert.Equal(t, "no-store", res.Header().Get("Cache-Control"), "who a player is changes when it renames")
}

func TestAFailedReadIsTheErrorNets(t *testing.T) {
	failure := errors.New("postgres is down")

	_, err := getAuthors(t, &stubQuery{err: failure}, adaID)

	assert.ErrorIs(t, err, failure)
}
