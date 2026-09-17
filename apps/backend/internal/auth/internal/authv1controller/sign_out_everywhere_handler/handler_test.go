package sign_out_everywhere_handler_test

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
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/authv1controller/sign_out_everywhere_handler"
)

type stubUseCase struct {
	setCookie string
	err       error
}

func (s stubUseCase) Execute(context.Context, string) (string, error) {
	return s.setCookie, s.err
}

func signOutEverywhere(useCase stubUseCase) (*connect.Response[authv1.SignOutEverywhereResponse], error) {
	res, err := sign_out_everywhere_handler.New(useCase).SignOutEverywhere(context.Background(), connect.NewRequest(&authv1.SignOutEverywhereRequest{}))
	if err != nil {
		return nil, fmt.Errorf("SignOutEverywhere failed: %w", err)
	}
	return res, nil
}

func TestSigningOutEverywhereAnswersTheClearedCookie(t *testing.T) {
	res, err := signOutEverywhere(stubUseCase{setCookie: "cp_sid=; Max-Age=0"})
	require.NoError(t, err)

	assert.Equal(t, "cp_sid=; Max-Age=0", res.Header().Get("Set-Cookie"))
	assert.Equal(t, "no-store", res.Header().Get("Cache-Control"))
}

func TestNoAccountIsUnauthenticatedAndAFailureIsNot(t *testing.T) {
	_, err := signOutEverywhere(stubUseCase{err: accounts.ErrNoAccount})
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))

	_, err = signOutEverywhere(stubUseCase{err: errors.New("postgres is down")})
	require.Error(t, err)
	assert.NotEqual(t, connect.CodeUnauthenticated, connect.CodeOf(err))
}
