package rpc_player_names_test

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/links"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/links/rpc_player_names"
)

var account = links.AccountID(uuid.MustParse("01a19416-ad77-72b3-8c34-8bbbaec97d8b"))

type player struct {
	answer *playerv1.GetPlayersResponse
	err    error
	asked  []string
}

func (p *player) GetPlayers(
	_ context.Context, req *connect.Request[playerv1.GetPlayersRequest],
) (*connect.Response[playerv1.GetPlayersResponse], error) {
	p.asked = req.Msg.GetAccountIds()
	if p.err != nil {
		return nil, p.err
	}
	return connect.NewResponse(p.answer), nil
}

func TestAPlayersUsernameIsItsName(t *testing.T) {
	asked := &player{answer: &playerv1.GetPlayersResponse{Players: []*playerv1.Player{
		{AccountId: account.String(), Name: "BraveFox42"},
	}}}

	name, err := rpc_player_names.New(asked).NameOf(t.Context(), account)

	require.NoError(t, err)
	assert.Equal(t, links.Name("BraveFox42"), name)
	assert.Equal(t, []string{account.String()}, asked.asked)
}

func TestAGuestHasNoName(t *testing.T) {
	name, err := rpc_player_names.New(&player{answer: &playerv1.GetPlayersResponse{Players: []*playerv1.Player{
		{AccountId: account.String(), Name: "guest_a1b2c3", Guest: true},
	}}}).NameOf(t.Context(), account)

	require.NoError(t, err)
	assert.Empty(t, name)
}

func TestAnAccountPlayerDoesNotShowHasNoName(t *testing.T) {
	name, err := rpc_player_names.New(&player{answer: &playerv1.GetPlayersResponse{}}).NameOf(t.Context(), account)

	require.NoError(t, err)
	assert.Empty(t, name)
}
