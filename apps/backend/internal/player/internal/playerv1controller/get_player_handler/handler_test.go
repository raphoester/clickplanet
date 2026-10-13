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
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/get_player_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/get_player_handler/player_query"
)

type stubQuery struct {
	answer *playerv1.GetPlayerResponse
	err    error
	asked  []players.AccountID
	named  []string
}

func (s *stubQuery) Player(_ context.Context, account players.AccountID) (*playerv1.GetPlayerResponse, error) {
	s.asked = append(s.asked, account)
	return s.answer, s.err
}

func (s *stubQuery) PlayerNamed(_ context.Context, name string) (*playerv1.GetPlayerResponse, error) {
	s.named = append(s.named, name)
	return s.answer, s.err
}

var ada = players.AccountID{15: 1}

func getPlayer(t *testing.T, query *stubQuery, req *playerv1.GetPlayerRequest) (*connect.Response[playerv1.GetPlayerResponse], error) {
	t.Helper()

	return get_player_handler.New(query).GetPlayer(t.Context(), connect.NewRequest(req)) //nolint:wrapcheck // the tests read the connect error.
}

func ofAccount(account players.AccountID) *playerv1.GetPlayerRequest {
	return &playerv1.GetPlayerRequest{AccountId: account.String()}
}

func TestThePlayerIsTheQuerysAndMayBeCached(t *testing.T) {
	query := &stubQuery{answer: &playerv1.GetPlayerResponse{Player: &playerv1.Player{Name: "Ada_L"}}}

	res, err := getPlayer(t, query, ofAccount(ada))

	require.NoError(t, err)
	assert.Equal(t, []players.AccountID{ada}, query.asked)
	assert.Empty(t, query.named)
	assert.True(t, proto.Equal(query.answer, res.Msg))
	assert.Equal(t, "public, max-age=10", res.Header().Get("Cache-Control"))
}

func TestAnAccountIsAskedForWhateverNameComesWithIt(t *testing.T) {
	query := &stubQuery{answer: &playerv1.GetPlayerResponse{}}

	_, err := getPlayer(t, query, &playerv1.GetPlayerRequest{AccountId: ada.String(), Name: "Bob"})

	require.NoError(t, err)
	assert.Equal(t, []players.AccountID{ada}, query.asked)
	assert.Empty(t, query.named)
}

func TestAClientFromBeforeAccountIDsIsAnsweredByName(t *testing.T) {
	query := &stubQuery{answer: &playerv1.GetPlayerResponse{Player: &playerv1.Player{Name: "Ada_L"}}}

	res, err := getPlayer(t, query, &playerv1.GetPlayerRequest{Name: "ada_l"})

	require.NoError(t, err)
	assert.Equal(t, []string{"ada_l"}, query.named)
	assert.Empty(t, query.asked)
	assert.True(t, proto.Equal(query.answer, res.Msg))
}

func TestAnIDThatIsNotAnAccountIsRefused(t *testing.T) {
	for _, id := range []string{"", "Ada_L", "00000000-0000-0000-0000-000000000000"} {
		query := &stubQuery{answer: &playerv1.GetPlayerResponse{}}

		_, err := getPlayer(t, query, &playerv1.GetPlayerRequest{AccountId: id})

		assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err), "%q", id)
		assert.Empty(t, query.asked, "%q", id)
		assert.Empty(t, query.named, "%q", id)
	}
}

func TestAnAccountWithNoUsernameIsNotFound(t *testing.T) {
	_, err := getPlayer(t, &stubQuery{err: player_query.ErrNoPlayer}, ofAccount(ada))

	assert.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
}

func TestAStoreFailureIsNotNotFound(t *testing.T) {
	_, err := getPlayer(t, &stubQuery{err: errors.New("postgres is down")}, ofAccount(ada))

	require.Error(t, err)
	assert.NotEqual(t, connect.CodeNotFound, connect.CodeOf(err))
	assert.NotEqual(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}
