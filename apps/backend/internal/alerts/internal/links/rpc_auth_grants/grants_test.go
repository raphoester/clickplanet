package rpc_auth_grants_test

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/links"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/links/rpc_auth_grants"
)

var account = links.AccountID(uuid.MustParse("01a19416-ad77-72b3-8c34-8bbbaec97d8b"))

type stubAuth struct {
	asked  []string
	answer *authv1.GetDiscordGrantResponse
	err    error
}

func (a *stubAuth) GetDiscordGrant(
	_ context.Context, req *connect.Request[authv1.GetDiscordGrantRequest],
) (*connect.Response[authv1.GetDiscordGrantResponse], error) {
	a.asked = append(a.asked, req.Msg.GetGrantId())
	if a.err != nil {
		return nil, a.err
	}
	return connect.NewResponse(a.answer), nil
}

func TestAuthAnswersWhatTheGrantJoinsWith(t *testing.T) {
	auth := &stubAuth{answer: &authv1.GetDiscordGrantResponse{
		AccountId: account.String(), DiscordUserId: "4242", AccessToken: "the-access-token",
	}}

	grant, err := rpc_auth_grants.New(auth).Grant(t.Context(), "the-grant")

	require.NoError(t, err)
	assert.Equal(t, links.Grant{Account: account, User: "4242", Token: "the-access-token"}, grant)
	assert.Equal(t, []string{"the-grant"}, auth.asked)
}

func TestAGrantAuthKeepsNoMoreIsNoGrant(t *testing.T) {
	_, err := rpc_auth_grants.New(&stubAuth{err: connect.NewError(connect.CodeNotFound, errors.New("expired"))}).
		Grant(t.Context(), "the-grant")

	require.ErrorIs(t, err, links.ErrNoGrant)
}

func TestAnAuthThatFailsIsAnErrorToTryAgain(t *testing.T) {
	for name, err := range map[string]error{
		"unreachable": connect.NewError(connect.CodeUnavailable, errors.New("connection refused")),
		"failing":     connect.NewError(connect.CodeInternal, errors.New("postgres is down")),
	} {
		t.Run(name, func(t *testing.T) {
			_, got := rpc_auth_grants.New(&stubAuth{err: err}).Grant(t.Context(), "the-grant")

			require.Error(t, got)
			assert.NotErrorIs(t, got, links.ErrNoGrant)
		})
	}
}

func TestAnAnswerTheServerCannotJoinWithIsAnError(t *testing.T) {
	for name, answer := range map[string]*authv1.GetDiscordGrantResponse{
		"no account":      {DiscordUserId: "4242", AccessToken: "the-access-token"},
		"no Discord user": {AccountId: account.String(), DiscordUserId: "not-an-id", AccessToken: "the-access-token"},
		"no token":        {AccountId: account.String(), DiscordUserId: "4242"},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := rpc_auth_grants.New(&stubAuth{answer: answer}).Grant(t.Context(), "the-grant")

			require.Error(t, err)
			assert.NotContains(t, err.Error(), "the-access-token")
		})
	}
}
