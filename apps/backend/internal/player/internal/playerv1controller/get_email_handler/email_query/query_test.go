package email_query_test

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/get_email_handler/email_query"
)

type stubAuth struct {
	email *string
	err   error
	asked []string
}

func (s *stubAuth) GetEmail(
	_ context.Context, req *connect.Request[authv1.GetEmailRequest],
) (*connect.Response[authv1.GetEmailResponse], error) {
	s.asked = append(s.asked, req.Msg.GetAccountId())
	if s.err != nil {
		return nil, s.err
	}
	return connect.NewResponse(&authv1.GetEmailResponse{Email: s.email}), nil
}

var ada = players.AccountID{15: 1}

func TestTheAddressIsAuthsForTheSameAccount(t *testing.T) {
	email := "ada@example.com"
	auth := &stubAuth{email: &email}

	answer, err := email_query.New(auth).Email(t.Context(), ada)

	require.NoError(t, err)
	assert.Equal(t, "ada@example.com", answer.GetEmail())
	assert.Equal(t, []string{ada.String()}, auth.asked)
}

func TestNoAddressAtAuthIsNoAddress(t *testing.T) {
	answer, err := email_query.New(&stubAuth{}).Email(t.Context(), ada)

	require.NoError(t, err)
	assert.Nil(t, answer.Email)
}

func TestAnAuthThatFailsFailsTheRead(t *testing.T) {
	_, err := email_query.New(&stubAuth{err: connect.NewError(connect.CodeUnavailable, assert.AnError)}).Email(t.Context(), ada)

	require.ErrorIs(t, err, assert.AnError)
}
