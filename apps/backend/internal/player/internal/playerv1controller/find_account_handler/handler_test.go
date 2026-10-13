package find_account_handler_test

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/find_account_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/find_account_handler/account_query"
)

type stubQuery struct {
	answer *playerv1.FindAccountResponse
	err    error
	asked  []string
}

func (s *stubQuery) Account(_ context.Context, name string) (*playerv1.FindAccountResponse, error) {
	s.asked = append(s.asked, name)
	return s.answer, s.err
}

func findAccount(t *testing.T, query *stubQuery, name string) (*connect.Response[playerv1.FindAccountResponse], error) {
	t.Helper()

	return find_account_handler.New(query).FindAccount(t.Context(), connect.NewRequest(&playerv1.FindAccountRequest{Name: name})) //nolint:wrapcheck // the tests read the connect error.
}

func TestTheAccountIsTheQuerysAndIsNeverCached(t *testing.T) {
	query := &stubQuery{answer: &playerv1.FindAccountResponse{AccountId: "00000000-0000-0000-0000-000000000001"}}

	res, err := findAccount(t, query, "ada_l")

	require.NoError(t, err)
	assert.Equal(t, []string{"ada_l"}, query.asked)
	assert.True(t, proto.Equal(query.answer, res.Msg))
	assert.Equal(t, "no-store", res.Header().Get("Cache-Control"))
}

func TestANameNoAccountIsShownUnderIsNotFound(t *testing.T) {
	_, err := findAccount(t, &stubQuery{err: account_query.ErrNoAccount}, "Bob")

	assert.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
}

func TestAStoreFailureIsNotNotFound(t *testing.T) {
	_, err := findAccount(t, &stubQuery{err: errors.New("postgres is down")}, "Ada_L")

	require.Error(t, err)
	assert.NotEqual(t, connect.CodeNotFound, connect.CodeOf(err))
}
