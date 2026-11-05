package rpc_player_accounts_test

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions/rpc_player_accounts"
)

type stubPlayer struct {
	asked  [][]string
	answer []*playerv1.Account
	err    error
}

func (p *stubPlayer) GetAccounts(
	_ context.Context, req *connect.Request[playerv1.GetAccountsRequest],
) (*connect.Response[playerv1.GetAccountsResponse], error) {
	p.asked = append(p.asked, req.Msg.GetAccountIds())
	if p.err != nil {
		return nil, p.err
	}
	return connect.NewResponse(&playerv1.GetAccountsResponse{Accounts: p.answer}), nil
}

var account = missions.AccountID{15: 7}

func TestPlayerSaysWhetherTheAccountIsSignedIn(t *testing.T) {
	for _, linked := range []bool{true, false} {
		player := &stubPlayer{answer: []*playerv1.Account{{AccountId: account.String(), Linked: linked}}}

		answer, err := rpc_player_accounts.New(player).Linked(t.Context(), account)

		require.NoError(t, err)
		assert.Equal(t, linked, answer)
		assert.Equal(t, [][]string{{account.String()}}, player.asked)
	}
}

func TestAnAccountPlayerDoesNotAnswerIsNotSignedIn(t *testing.T) {
	answer, err := rpc_player_accounts.New(&stubPlayer{}).Linked(t.Context(), account)

	require.NoError(t, err)
	assert.False(t, answer)
}

func TestAPlayerThatFailsIsAnError(t *testing.T) {
	_, err := rpc_player_accounts.New(&stubPlayer{err: errors.New("down")}).Linked(t.Context(), account)

	assert.Error(t, err)
}
