package get_accounts_handler_test

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/authv1controller/get_accounts_handler"
)

const accountID = "0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11"

type stubQuery struct {
	found *authv1.GetAccountsResponse
	err   error
	asked []accounts.AccountID
}

func (s *stubQuery) Accounts(_ context.Context, asked []accounts.AccountID) (*authv1.GetAccountsResponse, error) {
	s.asked = asked
	return s.found, s.err
}

func getAccounts(t *testing.T, query *stubQuery, ids ...string) (*connect.Response[authv1.GetAccountsResponse], error) {
	t.Helper()

	return get_accounts_handler.New(query).GetAccounts(t.Context(), //nolint:wrapcheck // the tests read the connect error.
		connect.NewRequest(&authv1.GetAccountsRequest{AccountIds: ids}))
}

func TestTheAccountsAreWhatTheQueryReadForTheIDs(t *testing.T) {
	account, err := accounts.AccountIDOf(accountID)
	require.NoError(t, err)
	found := &authv1.GetAccountsResponse{Accounts: []*authv1.Account{
		{AccountId: accountID, Linked: true, CreatedAtUnixMs: 1_788_000_000_000},
	}}
	query := &stubQuery{found: found}

	res, err := getAccounts(t, query, accountID)

	require.NoError(t, err)
	assert.Equal(t, []accounts.AccountID{account}, query.asked)
	assert.True(t, proto.Equal(found, res.Msg), "got %v", res.Msg)
}

func TestAnIDThatIsNotAnAccountIsInvalid(t *testing.T) {
	query := &stubQuery{}

	_, err := getAccounts(t, query, accountID, "not-an-id")

	assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
	assert.Nil(t, query.asked, "nothing is read")
}

func TestAFailedReadIsNotTheCallersFault(t *testing.T) {
	_, err := getAccounts(t, &stubQuery{err: errors.New("postgres is down")}, accountID)

	require.Error(t, err)
	assert.NotEqual(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}
