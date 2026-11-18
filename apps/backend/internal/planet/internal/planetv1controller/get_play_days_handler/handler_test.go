package get_play_days_handler_test

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_play_days_handler"
)

const (
	ada = "0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11"
	bob = "5a0c2f1e-7d3b-4c8a-9e6f-1b2d3c4e5f60"
)

type stubQuery struct {
	answer *planetv1.GetPlayDaysResponse
	asked  [][]ledger.AccountID
}

func (s *stubQuery) PlayDays(_ context.Context, accounts []ledger.AccountID) (*planetv1.GetPlayDaysResponse, error) {
	s.asked = append(s.asked, accounts)
	return s.answer, nil
}

func ask(t *testing.T, query *stubQuery, accounts ...string) (*connect.Response[planetv1.GetPlayDaysResponse], error) {
	t.Helper()

	return get_play_days_handler.New(query).GetPlayDays(t.Context(), //nolint:wrapcheck // the test reads the handler's own error.
		connect.NewRequest(&planetv1.GetPlayDaysRequest{AccountIds: accounts}))
}

func id(t *testing.T, account string) ledger.AccountID {
	t.Helper()

	parsed, err := ledger.AccountIDOf(account)
	require.NoError(t, err)
	return parsed
}

func TestTheAccountsAnswerIsTheQuerys(t *testing.T) {
	query := &stubQuery{answer: &planetv1.GetPlayDaysResponse{Accounts: []*planetv1.AccountPlayDays{
		{AccountId: ada, Days: []string{"2026-09-20"}},
	}}}

	res, err := ask(t, query, ada, bob)

	require.NoError(t, err)
	assert.True(t, proto.Equal(query.answer, res.Msg))
	assert.Equal(t, [][]ledger.AccountID{{id(t, ada), id(t, bob)}}, query.asked)
}

func TestNoAccountAskedReadsNothing(t *testing.T) {
	query := &stubQuery{}

	res, err := ask(t, query)

	require.NoError(t, err)
	assert.Empty(t, res.Msg.GetAccounts())
	assert.Empty(t, query.asked)
}

func TestAnIDThatIsNotAnAccountIsInvalidAndReadsNothing(t *testing.T) {
	query := &stubQuery{}

	for _, account := range []string{"", "nope", "00000000-0000-0000-0000-000000000000"} {
		_, err := ask(t, query, ada, account)

		assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err), "%q", account)
	}
	assert.Empty(t, query.asked)
}

func TestMoreThanAPageOfAccountsIsInvalidAndReadsNothing(t *testing.T) {
	query := &stubQuery{answer: &planetv1.GetPlayDaysResponse{}}
	page := make([]string, get_play_days_handler.MaxAccounts)
	for i := range page {
		page[i] = ada
	}

	_, err := ask(t, query, page...)
	require.NoError(t, err)

	_, err = ask(t, query, append(page, bob)...)
	assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
	assert.Len(t, query.asked, 1)
}
