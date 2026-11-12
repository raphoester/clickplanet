package get_player_by_name_handler_test

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/get_player_by_name_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/get_player_by_name_handler/holder_query"
)

type stubQuery struct {
	answer *playerv1.GetPlayerByNameResponse
	err    error
	asked  []string
}

func (s *stubQuery) Holder(_ context.Context, folded string) (*playerv1.GetPlayerByNameResponse, error) {
	s.asked = append(s.asked, folded)
	return s.answer, s.err
}

func getPlayerByName(t *testing.T, query *stubQuery, name string) (*connect.Response[playerv1.GetPlayerByNameResponse], error) {
	t.Helper()

	return get_player_by_name_handler.New(query).GetPlayerByName(t.Context(), //nolint:wrapcheck // the tests read the connect error.
		connect.NewRequest(&playerv1.GetPlayerByNameRequest{Name: name}))
}

func TestTheHolderIsTheQuerysAndMayBeCached(t *testing.T) {
	query := &stubQuery{answer: &playerv1.GetPlayerByNameResponse{Player: &playerv1.Player{AccountId: "ada", Name: "Ada_L"}}}

	res, err := getPlayerByName(t, query, "Ada_L")

	require.NoError(t, err)
	assert.True(t, proto.Equal(query.answer, res.Msg))
	assert.Equal(t, "public, max-age=10", res.Header().Get("Cache-Control"))
}

func TestTheNameIsAskedFoldedAsSetNameKeepsIt(t *testing.T) {
	for asked, folded := range map[string]string{
		"Ada_L":        "ada_l",
		"Ada Lovelace": "ada lovelace",
		"STRASSE":      "strasse",
		"Straße":       "strasse",
		"Ａｄａ":          "ada",
		"Émile":        "émile",
		" Ada":         " ada",
		"Ada-Lovelace": "ada-lovelace",
		"МАША Котова":  "маша котова",
	} {
		query := &stubQuery{answer: &playerv1.GetPlayerByNameResponse{}}

		_, err := getPlayerByName(t, query, asked)

		require.NoError(t, err, "%q", asked)
		assert.Equal(t, []string{folded}, query.asked, "%q", asked)
	}
}

func TestNoNameIsRefused(t *testing.T) {
	query := &stubQuery{answer: &playerv1.GetPlayerByNameResponse{}}

	_, err := getPlayerByName(t, query, "")

	assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
	assert.Empty(t, query.asked)
}

func TestANameNobodyHoldsIsNotFound(t *testing.T) {
	_, err := getPlayerByName(t, &stubQuery{err: holder_query.ErrNoHolder}, "Nobody")

	assert.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
}

func TestAStoreFailureIsNotNotFound(t *testing.T) {
	_, err := getPlayerByName(t, &stubQuery{err: errors.New("postgres is down")}, "Ada_L")

	require.Error(t, err)
	assert.NotEqual(t, connect.CodeNotFound, connect.CodeOf(err))
	assert.NotEqual(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}
