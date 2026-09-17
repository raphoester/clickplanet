package get_account_handler_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/authv1controller/get_account_handler"
)

const accountID = "0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11"

type stubUseCase struct {
	account *accounts.Account
	err     error
	asked   []accounts.AccountID
}

func (s *stubUseCase) Execute(_ context.Context, account accounts.AccountID) (*accounts.Account, error) {
	s.asked = append(s.asked, account)
	return s.account, s.err
}

func getAccount(t *testing.T, useCase *stubUseCase, id string) (*authv1.GetAccountResponse, error) {
	t.Helper()

	res, err := get_account_handler.New(useCase).GetAccount(t.Context(), connect.NewRequest(&authv1.GetAccountRequest{AccountId: id}))
	if err != nil {
		return nil, fmt.Errorf("GetAccount failed: %w", err)
	}
	return res.Msg, nil
}

func TestALinkedAccountIsLinked(t *testing.T) {
	useCase := &stubUseCase{account: &accounts.Account{Identities: []accounts.Identity{{Provider: "google"}}}}

	res, err := getAccount(t, useCase, accountID)

	require.NoError(t, err)
	assert.True(t, res.GetLinked())
	want, err := accounts.AccountIDOf(accountID)
	require.NoError(t, err)
	assert.Equal(t, []accounts.AccountID{want}, useCase.asked)
}

func TestTheAccountSaysWhenItWasMade(t *testing.T) {
	createdAt := time.Date(2026, 9, 1, 8, 30, 0, 0, time.UTC)

	res, err := getAccount(t, &stubUseCase{account: &accounts.Account{CreatedAt: createdAt}}, accountID)

	require.NoError(t, err)
	assert.Equal(t, createdAt.UnixMilli(), res.GetCreatedAtUnixMs())
}

func TestAGuestIsNotLinked(t *testing.T) {
	res, err := getAccount(t, &stubUseCase{account: &accounts.Account{}}, accountID)

	require.NoError(t, err)
	assert.False(t, res.GetLinked())
}

func TestAnUnknownAccountIsNotLinkedAndNotAnError(t *testing.T) {
	res, err := getAccount(t, &stubUseCase{err: accounts.ErrAccountNotFound}, accountID)

	require.NoError(t, err)
	assert.False(t, res.GetLinked())
	assert.Zero(t, res.GetCreatedAtUnixMs())
}

func TestAnIDThatIsNotAnAccountIsNotLinkedAndNobodyIsAsked(t *testing.T) {
	for _, id := range []string{"", "not-a-uuid", "00000000-0000-0000-0000-000000000000"} {
		useCase := &stubUseCase{account: &accounts.Account{Identities: []accounts.Identity{{Provider: "google"}}}}

		res, err := getAccount(t, useCase, id)

		require.NoError(t, err, "%q", id)
		assert.False(t, res.GetLinked(), "%q", id)
		assert.Empty(t, useCase.asked, "%q", id)
	}
}

func TestAStoreFailureIsAnError(t *testing.T) {
	_, err := getAccount(t, &stubUseCase{err: errors.New("postgres is down")}, accountID)

	assert.Error(t, err)
}
