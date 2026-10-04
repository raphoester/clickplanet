package get_caller_handler_test

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
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/authv1controller/get_caller_handler"
)

type stubUseCase struct {
	account accounts.AccountID
	err     error
	asked   []string
}

func (s *stubUseCase) Execute(_ context.Context, cookieHeader string) (accounts.AccountID, error) {
	s.asked = append(s.asked, cookieHeader)
	return s.account, s.err
}

func getCaller(t *testing.T, useCase *stubUseCase, cookie string) (*authv1.GetCallerResponse, error) {
	t.Helper()

	res, err := get_caller_handler.New(useCase).GetCaller(t.Context(), connect.NewRequest(&authv1.GetCallerRequest{Cookie: cookie}))
	if err != nil {
		return nil, fmt.Errorf("GetCaller failed: %w", err)
	}
	return res.Msg, nil
}

func TestTheCookieIsAnsweredWithItsAccount(t *testing.T) {
	useCase := &stubUseCase{account: accounts.AccountID{15: 1}}

	res, err := getCaller(t, useCase, "cp_sid=token-1")

	require.NoError(t, err)
	assert.Equal(t, accounts.AccountID{15: 1}.String(), res.GetAccountId())
	assert.Equal(t, []string{"cp_sid=token-1"}, useCase.asked)
}

func TestNoAccountIsAnEmptyAnswerAndNotAnError(t *testing.T) {
	res, err := getCaller(t, &stubUseCase{err: fmt.Errorf("wrapped: %w", accounts.ErrNoAccount)}, "cp_sid=made-up")

	require.NoError(t, err)
	assert.Empty(t, res.GetAccountId())
}

func TestAStoreFailureIsAnError(t *testing.T) {
	_, err := getCaller(t, &stubUseCase{err: errors.New("postgres is down")}, "cp_sid=token-1")

	assert.Error(t, err)
}
