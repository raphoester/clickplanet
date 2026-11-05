package search_players_handler_test

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/search_players_handler"
)

type search struct {
	folded string
	limit  int
}

type stubQuery struct {
	answer *playerv1.SearchPlayersResponse
	err    error
	asked  []search
}

func (s *stubQuery) Search(_ context.Context, folded string, limit int) (*playerv1.SearchPlayersResponse, error) {
	s.asked = append(s.asked, search{folded: folded, limit: limit})
	return s.answer, s.err
}

func found() *stubQuery {
	return &stubQuery{answer: &playerv1.SearchPlayersResponse{Total: 3, Players: []*playerv1.PlayerMatch{
		{AccountId: "0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11", Name: "EIRE"},
		{AccountId: "5e0c1b2a-3d4e-4f60-8a71-9b2c3d4e5f60", Name: "EIRE1"},
	}}}
}

func searchPlayers(
	t *testing.T, query *stubQuery, req *playerv1.SearchPlayersRequest,
) (*connect.Response[playerv1.SearchPlayersResponse], error) {
	t.Helper()
	return search_players_handler.New(query).SearchPlayers(t.Context(), connect.NewRequest(req)) //nolint:wrapcheck // the test reads the handler's own error.
}

func TestTheNameIsFoldedAndTheMatchesAreTheQuerys(t *testing.T) {
	query := found()

	res, err := searchPlayers(t, query, &playerv1.SearchPlayersRequest{Name: "  Éire "})
	require.NoError(t, err)

	assert.Equal(t, []search{{folded: "éire", limit: 20}}, query.asked)
	assert.True(t, proto.Equal(query.answer, res.Msg))
}

func TestAnEmptyNameIsRefusedAndReadsNothing(t *testing.T) {
	for _, name := range []string{"", "   "} {
		query := found()
		_, err := searchPlayers(t, query, &playerv1.SearchPlayersRequest{Name: name})
		assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err), name)
		assert.Empty(t, query.asked, name)
	}
}

func TestTheLimitAskedIsCapped(t *testing.T) {
	for asked, want := range map[uint32]int{7: 7, 500: 500, 10_000: 500} {
		query := found()
		_, err := searchPlayers(t, query, &playerv1.SearchPlayersRequest{Name: "eire", Limit: asked})
		require.NoError(t, err)
		assert.Equal(t, []search{{folded: "eire", limit: want}}, query.asked, asked)
	}
}

func TestTheAnswerIsNeverStored(t *testing.T) {
	res, err := searchPlayers(t, found(), &playerv1.SearchPlayersRequest{Name: "eire"})
	require.NoError(t, err)
	assert.Equal(t, "no-store", res.Header().Get("Cache-Control"), "who a player is changes when it renames")
}

func TestAFailedReadIsTheErrorNets(t *testing.T) {
	failure := errors.New("postgres is down")
	query := found()
	query.err = failure
	_, err := searchPlayers(t, query, &playerv1.SearchPlayersRequest{Name: "eire"})
	assert.ErrorIs(t, err, failure)
}
