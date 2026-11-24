package rpc_accounts_test

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes/rpc_accounts"
)

type stubPlayer struct {
	asked [][]string
}

func (s *stubPlayer) GetAccounts(
	_ context.Context, req *connect.Request[playerv1.GetAccountsRequest],
) (*connect.Response[playerv1.GetAccountsResponse], error) {
	s.asked = append(s.asked, req.Msg.GetAccountIds())
	return connect.NewResponse(&playerv1.GetAccountsResponse{Accounts: []*playerv1.Account{
		{AccountId: notes.AccountID{15: 1}.String(), Linked: true},
		{AccountId: notes.AccountID{15: 2}.String(), Linked: false},
	}}), nil
}

func TestTheLinkedAccountsAreTheOnesPlayerSaysSignedIn(t *testing.T) {
	player := &stubPlayer{}
	asked := []notes.AccountID{{15: 1}, {15: 2}, {15: 3}}

	linked, err := rpc_accounts.New(player).Linked(t.Context(), asked)

	require.NoError(t, err)
	assert.True(t, linked.Contains(notes.AccountID{15: 1}))
	assert.False(t, linked.Contains(notes.AccountID{15: 2}))
	assert.False(t, linked.Contains(notes.AccountID{15: 3}), "an account player does not know is no signed-in one")
	assert.Len(t, player.asked, 1)
}

func TestNoAccountAsksNothing(t *testing.T) {
	player := &stubPlayer{}

	linked, err := rpc_accounts.New(player).Linked(t.Context(), nil)

	require.NoError(t, err)
	assert.True(t, linked.Empty())
	assert.Empty(t, player.asked)
}
