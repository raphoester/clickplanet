package admin_get_player_handler_test

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
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/admin_get_player_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/admin_get_player_handler/admin_player_query"
)

type stubQuery struct {
	answer *playerv1.AdminGetPlayerResponse
	err    error
	asked  []players.AccountID
}

func (s *stubQuery) Player(_ context.Context, account players.AccountID) (*playerv1.AdminGetPlayerResponse, error) {
	s.asked = append(s.asked, account)
	return s.answer, s.err
}

var ada = players.AccountID{15: 1}

func getPlayer(t *testing.T, query *stubQuery, id string) (*connect.Response[playerv1.AdminGetPlayerResponse], error) {
	t.Helper()

	return admin_get_player_handler.New(query).GetPlayer( //nolint:wrapcheck // the tests read the connect error.
		t.Context(), connect.NewRequest(&playerv1.AdminGetPlayerRequest{AccountId: id}),
	)
}

func TestThePlayerIsTheQuerysAndIsNeverCached(t *testing.T) {
	query := &stubQuery{answer: &playerv1.AdminGetPlayerResponse{Name: "guest_a1b2c3", Guest: true}}

	res, err := getPlayer(t, query, ada.String())

	require.NoError(t, err)
	assert.Equal(t, []players.AccountID{ada}, query.asked)
	assert.True(t, proto.Equal(query.answer, res.Msg))
	assert.Equal(t, "no-store", res.Header().Get("Cache-Control"))
}

func TestAnIDThatIsNotAnAccountIsRefused(t *testing.T) {
	for _, id := range []string{"", "Ada_L", "9.9.9.9", "00000000-0000-0000-0000-000000000000"} {
		query := &stubQuery{answer: &playerv1.AdminGetPlayerResponse{}}

		_, err := getPlayer(t, query, id)

		assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err), "%q", id)
		assert.Empty(t, query.asked, "%q", id)
	}
}

func TestAnAccountTheGameKeepsNothingAboutIsNotFound(t *testing.T) {
	_, err := getPlayer(t, &stubQuery{err: admin_player_query.ErrNoPlayer}, ada.String())

	assert.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
}

func TestAFailureIsLeftToTheErrorNet(t *testing.T) {
	cause := errors.New("postgres is down")

	_, err := getPlayer(t, &stubQuery{err: cause}, ada.String())

	require.ErrorIs(t, err, cause)
	assert.Equal(t, connect.CodeUnknown, connect.CodeOf(err))
}
