package get_bombs_and_fortifies_handler_test

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_bombs_and_fortifies_handler"
)

const ada = "0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11"

type stubQuery struct {
	answer *planetv1.GetBombsAndFortifiesResponse
	asked  [][]ledger.AccountID
}

func (s *stubQuery) BombsAndFortifies(_ context.Context, accounts []ledger.AccountID) (*planetv1.GetBombsAndFortifiesResponse, error) {
	s.asked = append(s.asked, accounts)
	return s.answer, nil
}

func ask(t *testing.T, query *stubQuery, accounts ...string) (*connect.Response[planetv1.GetBombsAndFortifiesResponse], error) {
	t.Helper()

	return get_bombs_and_fortifies_handler.New(query).GetBombsAndFortifies(t.Context(), //nolint:wrapcheck // the test reads the handler's own error.
		connect.NewRequest(&planetv1.GetBombsAndFortifiesRequest{AccountIds: accounts}))
}

func TestTheAccountsAskedAreAnsweredByTheQuery(t *testing.T) {
	query := &stubQuery{answer: &planetv1.GetBombsAndFortifiesResponse{Accounts: []*planetv1.BombsAndFortifies{{AccountId: ada, BombsDropped: 2}}}}

	res, err := ask(t, query, ada)

	require.NoError(t, err)
	assert.True(t, proto.Equal(query.answer, res.Msg))
	account, err := ledger.AccountIDOf(ada)
	require.NoError(t, err)
	assert.Equal(t, [][]ledger.AccountID{{account}}, query.asked)
}

func TestAnIDThatIsNotAnAccountIsInvalidAndReadsNothing(t *testing.T) {
	query := &stubQuery{}

	_, err := ask(t, query, ada, "nope")

	assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
	assert.Empty(t, query.asked)
}
