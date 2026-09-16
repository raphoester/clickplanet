package resolve_account_handler_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts/usecases/resolve_account_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/authv1controller/resolve_account_handler"
)

type stubUseCase struct {
	out   *resolve_account_usecase.Out
	err   error
	asked []resolve_account_usecase.In
}

func (s *stubUseCase) Execute(_ context.Context, in resolve_account_usecase.In) (*resolve_account_usecase.Out, error) {
	s.asked = append(s.asked, in)
	return s.out, s.err
}

func resolve(useCase *stubUseCase) (*authv1.ResolveAccountResponse, error) {
	res, err := resolve_account_handler.New(useCase).ResolveAccount(context.Background(),
		connect.NewRequest(&authv1.ResolveAccountRequest{CookieHeader: "cp_sid=abc", Create: true}))
	if err != nil {
		return nil, fmt.Errorf("ResolveAccount failed: %w", err)
	}
	return res.Msg, nil
}

func TestTheRequestReachesTheUseCaseAndTheAccountComesBack(t *testing.T) {
	account := uuid.MustParse("01926c6e-7a4b-7c3d-8e9f-0a1b2c3d4e5f")
	useCase := &stubUseCase{out: &resolve_account_usecase.Out{Account: account, SetCookie: "cp_sid=new"}}

	res, err := resolve(useCase)
	require.NoError(t, err)

	assert.Equal(t, []resolve_account_usecase.In{{CookieHeader: "cp_sid=abc", Create: true}}, useCase.asked)
	assert.Equal(t, account.String(), res.GetAccountId())
	assert.Equal(t, "cp_sid=new", res.GetSetCookie())
}

func TestNoAccountIsAnEmptyAnswerNotAnError(t *testing.T) {
	res, err := resolve(&stubUseCase{err: accounts.ErrNoAccount})
	require.NoError(t, err)

	assert.Empty(t, res.GetAccountId())
	assert.Empty(t, res.GetSetCookie())
}

func TestAFailureIsAnError(t *testing.T) {
	_, err := resolve(&stubUseCase{err: errors.New("postgres is down")})

	assert.Error(t, err)
}
