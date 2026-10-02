package get_accounts_handler_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/authv1controller/get_accounts_handler"
)

const accountID = "0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11"

type stubUseCase struct {
	found []*accounts.Account
	err   error
	asked []accounts.AccountID
}

func (s *stubUseCase) Execute(_ context.Context, asked []accounts.AccountID) ([]*accounts.Account, error) {
	s.asked = asked
	return s.found, s.err
}

func getAccounts(t *testing.T, useCase *stubUseCase, ids ...string) (*connect.Response[authv1.GetAccountsResponse], error) {
	t.Helper()

	return get_accounts_handler.New(useCase).GetAccounts(t.Context(), //nolint:wrapcheck // the tests read the connect error.
		connect.NewRequest(&authv1.GetAccountsRequest{AccountIds: ids}))
}

func TestEachKnownAccountIsAnsweredWithWhetherItIsLinkedAndWhenItWasMade(t *testing.T) {
	account, err := accounts.AccountIDOf(accountID)
	require.NoError(t, err)
	createdAt := time.Date(2026, 9, 1, 8, 30, 0, 0, time.UTC)
	useCase := &stubUseCase{found: []*accounts.Account{
		{ID: account, CreatedAt: createdAt, Identities: []accounts.Identity{{Provider: "google"}}},
	}}

	res, err := getAccounts(t, useCase, accountID)

	require.NoError(t, err)
	assert.Equal(t, []accounts.AccountID{account}, useCase.asked)
	require.Len(t, res.Msg.GetAccounts(), 1)
	assert.Equal(t, accountID, res.Msg.GetAccounts()[0].GetAccountId())
	assert.True(t, res.Msg.GetAccounts()[0].GetLinked())
	assert.Equal(t, createdAt.UnixMilli(), res.Msg.GetAccounts()[0].GetCreatedAtUnixMs())
}

func TestAnIDThatIsNotAnAccountIsInvalid(t *testing.T) {
	useCase := &stubUseCase{}

	_, err := getAccounts(t, useCase, accountID, "not-an-id")

	assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
	assert.Nil(t, useCase.asked, "nothing is read")
}

func TestAStoreFailureIsNotTheCallersFault(t *testing.T) {
	_, err := getAccounts(t, &stubUseCase{err: errors.New("postgres is down")}, accountID)

	require.Error(t, err)
	assert.NotEqual(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}
