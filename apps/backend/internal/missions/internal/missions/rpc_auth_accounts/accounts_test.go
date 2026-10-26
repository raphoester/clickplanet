package rpc_auth_accounts_test

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions/rpc_auth_accounts"
)

type stubAuth struct {
	asked  []string
	linked bool
	err    error
}

func (a *stubAuth) GetAccount(
	_ context.Context, req *connect.Request[authv1.GetAccountRequest],
) (*connect.Response[authv1.GetAccountResponse], error) {
	a.asked = append(a.asked, req.Msg.GetAccountId())
	if a.err != nil {
		return nil, a.err
	}
	return connect.NewResponse(&authv1.GetAccountResponse{Linked: a.linked}), nil
}

var account = missions.AccountID{15: 7}

func TestAuthSaysWhetherTheAccountIsSignedIn(t *testing.T) {
	for _, linked := range []bool{true, false} {
		auth := &stubAuth{linked: linked}

		answer, err := rpc_auth_accounts.New(auth).Linked(t.Context(), account)

		require.NoError(t, err)
		assert.Equal(t, linked, answer)
		assert.Equal(t, []string{account.String()}, auth.asked)
	}
}

func TestAnAuthThatFailsIsAnError(t *testing.T) {
	_, err := rpc_auth_accounts.New(&stubAuth{err: errors.New("down")}).Linked(t.Context(), account)

	assert.Error(t, err)
}
