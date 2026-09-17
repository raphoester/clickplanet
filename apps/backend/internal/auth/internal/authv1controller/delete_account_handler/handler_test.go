package delete_account_handler_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts/usecases/delete_account_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/authv1controller/delete_account_handler"
)

type stubUseCase struct {
	out *delete_account_usecase.Out
	err error
}

func (s stubUseCase) Execute(context.Context, string) (*delete_account_usecase.Out, error) {
	return s.out, s.err
}

func deleteAccount(useCase stubUseCase) (*connect.Response[authv1.DeleteAccountResponse], error) {
	res, err := delete_account_handler.New(useCase).DeleteAccount(context.Background(), connect.NewRequest(&authv1.DeleteAccountRequest{}))
	if err != nil {
		return nil, fmt.Errorf("DeleteAccount failed: %w", err)
	}
	return res, nil
}

func TestADeletedAccountAnswersTheClearedCookie(t *testing.T) {
	res, err := deleteAccount(stubUseCase{out: &delete_account_usecase.Out{Account: accounts.AccountID{15: 1}, SetCookie: "cp_sid=; Max-Age=0"}})
	require.NoError(t, err)

	assert.Equal(t, "cp_sid=; Max-Age=0", res.Header().Get("Set-Cookie"))
	assert.Equal(t, "no-store", res.Header().Get("Cache-Control"))
}

func TestNoAccountIsUnauthenticatedAndAFailureIsNot(t *testing.T) {
	_, err := deleteAccount(stubUseCase{err: accounts.ErrNoAccount})
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))

	_, err = deleteAccount(stubUseCase{err: errors.New("postgres is down")})
	require.Error(t, err)
	assert.NotEqual(t, connect.CodeUnauthenticated, connect.CodeOf(err))
}
