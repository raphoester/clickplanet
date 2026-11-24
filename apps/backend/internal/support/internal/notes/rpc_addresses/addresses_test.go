package rpc_addresses_test

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpmail"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes/rpc_addresses"
)

type stubPlayer struct {
	email *string
	err   error
	asked []string
}

func (s *stubPlayer) GetEmail(
	_ context.Context, req *connect.Request[playerv1.GetEmailRequest],
) (*connect.Response[playerv1.GetEmailResponse], error) {
	s.asked = append(s.asked, req.Msg.GetAccountId())
	if s.err != nil {
		return nil, s.err
	}
	return connect.NewResponse(&playerv1.GetEmailResponse{Email: s.email}), nil
}

var ada = notes.AccountID{15: 1}

func held(email string) *string {
	return &email
}

func TestTheAddressIsThePlayersOneForTheAccount(t *testing.T) {
	player := &stubPlayer{email: held("Ada@Example.com")}

	address, err := rpc_addresses.New(player).Address(t.Context(), ada)

	require.NoError(t, err)
	assert.Equal(t, cpmail.Address("ada@example.com"), address)
	assert.Equal(t, []string{ada.String()}, player.asked)
}

func TestNoAddressOrOneMailCannotGoToIsNone(t *testing.T) {
	for name, player := range map[string]*stubPlayer{
		"none":       {},
		"not usable": {email: held("ada@localhost")},
	} {
		_, err := rpc_addresses.New(player).Address(t.Context(), ada)

		require.ErrorIs(t, err, notes.ErrNoReplyAddress, name)
	}
}

func TestAPlayerThatFailsIsAFailureAndNotNoAddress(t *testing.T) {
	_, err := rpc_addresses.New(&stubPlayer{err: connect.NewError(connect.CodeUnavailable, assert.AnError)}).Address(t.Context(), ada)

	require.ErrorIs(t, err, assert.AnError)
	assert.NotErrorIs(t, err, notes.ErrNoReplyAddress)
}

func TestTheAddressesOfAPageOfAccountsLeaveOutTheOnesWithNone(t *testing.T) {
	held, err := rpc_addresses.New(&stubPlayer{email: held("ada@example.com")}).Addresses(t.Context(), []notes.AccountID{ada})
	require.NoError(t, err)
	assert.Equal(t, map[notes.AccountID]cpmail.Address{ada: "ada@example.com"}, held)

	none, err := rpc_addresses.New(&stubPlayer{}).Addresses(t.Context(), []notes.AccountID{ada})
	require.NoError(t, err)
	assert.Empty(t, none)

	_, err = rpc_addresses.New(&stubPlayer{err: connect.NewError(connect.CodeUnavailable, assert.AnError)}).Addresses(t.Context(), []notes.AccountID{ada})
	require.ErrorIs(t, err, assert.AnError)
}
