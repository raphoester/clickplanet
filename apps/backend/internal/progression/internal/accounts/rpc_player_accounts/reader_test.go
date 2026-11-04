package rpc_player_accounts_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/accounts/rpc_player_accounts"
)

var (
	ada   = accounts.AccountID{15: 1}
	guest = accounts.AccountID{15: 2}
	made  = time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC)
)

type stubPlayer struct {
	answer *playerv1.GetAccountsResponse
	err    error
	asked  [][]string
}

func (s *stubPlayer) GetAccounts(
	_ context.Context, req *connect.Request[playerv1.GetAccountsRequest],
) (*connect.Response[playerv1.GetAccountsResponse], error) {
	s.asked = append(s.asked, req.Msg.GetAccountIds())
	if s.err != nil {
		return nil, s.err
	}
	return connect.NewResponse(s.answer), nil
}

func TestEachAccountIsWhatPlayerSaysOfIt(t *testing.T) {
	player := &stubPlayer{answer: &playerv1.GetAccountsResponse{Accounts: []*playerv1.Account{
		{AccountId: ada.String(), Linked: true, CreatedAtUnixMs: made.UnixMilli()},
		{AccountId: guest.String()},
	}}}

	known, err := rpc_player_accounts.New(player).Accounts(t.Context(), []accounts.AccountID{ada, guest})

	require.NoError(t, err)
	assert.Equal(t, map[accounts.AccountID]accounts.Account{
		ada:   accounts.AccountOf(true, made),
		guest: accounts.AccountOf(false, time.Time{}),
	}, known)
	assert.Equal(t, [][]string{{ada.String(), guest.String()}}, player.asked)
}

func TestAnAccountPlayerDoesNotAnswerIsNotLinkedAndHasNoDate(t *testing.T) {
	account, err := rpc_player_accounts.New(&stubPlayer{answer: &playerv1.GetAccountsResponse{}}).Account(t.Context(), ada)

	require.NoError(t, err)
	assert.Equal(t, accounts.Account{}, account)
}

func TestNoAccountAsksNothing(t *testing.T) {
	player := &stubPlayer{}

	known, err := rpc_player_accounts.New(player).Accounts(t.Context(), nil)

	require.NoError(t, err)
	assert.Empty(t, known)
	assert.Empty(t, player.asked)
}

func TestAFailureToAskPlayerIsAnError(t *testing.T) {
	failure := errors.New("player is down")

	_, err := rpc_player_accounts.New(&stubPlayer{err: failure}).Account(t.Context(), ada)

	assert.ErrorIs(t, err, failure)
}

func TestAnAnswerThatIsNotAnAccountIsAnError(t *testing.T) {
	player := &stubPlayer{answer: &playerv1.GetAccountsResponse{Accounts: []*playerv1.Account{{AccountId: "nope"}}}}

	_, err := rpc_player_accounts.New(player).Accounts(t.Context(), []accounts.AccountID{ada})

	assert.ErrorIs(t, err, accounts.ErrInvalidAccount)
}
