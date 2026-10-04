package get_account_handler_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/authv1controller/get_account_handler"
)

const accountID = "0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11"

type stubQuery struct {
	account *authv1.GetAccountResponse
	err     error
	asked   []accounts.AccountID
}

func (s *stubQuery) Account(_ context.Context, account accounts.AccountID) (*authv1.GetAccountResponse, error) {
	s.asked = append(s.asked, account)
	return s.account, s.err
}

func getAccount(t *testing.T, query *stubQuery, id string) (*authv1.GetAccountResponse, error) {
	t.Helper()

	res, err := get_account_handler.New(query).GetAccount(t.Context(), connect.NewRequest(&authv1.GetAccountRequest{AccountId: id}))
	if err != nil {
		return nil, fmt.Errorf("GetAccount failed: %w", err)
	}
	return res.Msg, nil
}

func TestTheAccountIsWhatTheQueryReadForTheID(t *testing.T) {
	answer := &authv1.GetAccountResponse{Linked: true, CreatedAtUnixMs: 1_788_000_000_000}
	query := &stubQuery{account: answer}

	res, err := getAccount(t, query, accountID)

	require.NoError(t, err)
	assert.True(t, proto.Equal(answer, res), "got %v", res)
	want, err := accounts.AccountIDOf(accountID)
	require.NoError(t, err)
	assert.Equal(t, []accounts.AccountID{want}, query.asked)
}

func TestAnIDThatIsNotAnAccountIsNotLinkedAndNobodyIsAsked(t *testing.T) {
	for _, id := range []string{"", "not-a-uuid", "00000000-0000-0000-0000-000000000000"} {
		query := &stubQuery{account: &authv1.GetAccountResponse{Linked: true}}

		res, err := getAccount(t, query, id)

		require.NoError(t, err, "%q", id)
		assert.False(t, res.GetLinked(), "%q", id)
		assert.Empty(t, query.asked, "%q", id)
	}
}

func TestAFailedReadIsAnError(t *testing.T) {
	_, err := getAccount(t, &stubQuery{err: errors.New("postgres is down")}, accountID)

	assert.Error(t, err)
}
