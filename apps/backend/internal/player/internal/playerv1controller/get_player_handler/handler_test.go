package get_player_handler_test

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/get_player_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/get_player_handler/player_query"
)

type stubQuery struct {
	answer *playerv1.GetPlayerResponse
	err    error
	asked  []string
}

func (s *stubQuery) Player(_ context.Context, name string) (*playerv1.GetPlayerResponse, error) {
	s.asked = append(s.asked, name)
	return s.answer, s.err
}

func getPlayer(t *testing.T, query *stubQuery, name string) (*connect.Response[playerv1.GetPlayerResponse], error) {
	t.Helper()

	return get_player_handler.New(query).GetPlayer(t.Context(), connect.NewRequest(&playerv1.GetPlayerRequest{Name: name})) //nolint:wrapcheck // the tests read the connect error.
}

func TestThePlayerIsTheQuerysAndMayBeCached(t *testing.T) {
	query := &stubQuery{answer: &playerv1.GetPlayerResponse{Player: &playerv1.Player{Name: "Ada_L"}}}

	res, err := getPlayer(t, query, "ada_l")

	require.NoError(t, err)
	assert.Equal(t, []string{"ada_l"}, query.asked)
	assert.True(t, proto.Equal(query.answer, res.Msg))
	assert.Equal(t, "public, max-age=10", res.Header().Get("Cache-Control"))
}

func TestANameNobodyHoldsIsNotFound(t *testing.T) {
	_, err := getPlayer(t, &stubQuery{err: player_query.ErrNoPlayer}, "Bob")

	assert.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
}

func TestAStoreFailureIsNotNotFound(t *testing.T) {
	_, err := getPlayer(t, &stubQuery{err: errors.New("postgres is down")}, "Ada_L")

	require.Error(t, err)
	assert.NotEqual(t, connect.CodeNotFound, connect.CodeOf(err))
}
