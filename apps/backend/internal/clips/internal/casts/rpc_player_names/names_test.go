package rpc_player_names_test

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/casts"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/casts/rpc_player_names"
)

type player struct {
	authors []*playerv1.Author
	asked   []string
}

func (p *player) GetAuthors(
	_ context.Context,
	req *connect.Request[playerv1.GetAuthorsRequest],
) (*connect.Response[playerv1.GetAuthorsResponse], error) {
	p.asked = append(p.asked, req.Msg.GetAccountIds()...)
	return connect.NewResponse(&playerv1.GetAuthorsResponse{Authors: p.authors}), nil
}

func account(t *testing.T, id string) casts.AccountID {
	t.Helper()
	account, err := casts.AccountIDOf(id)
	require.NoError(t, err)
	return account
}

func TestOnlyANameItsPlayerLetsTheClipsShowIsShown(t *testing.T) {
	const ada, bob, guest = "0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a01", "0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a02", "0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a03"
	module := &player{authors: []*playerv1.Author{
		{AccountId: ada, Name: "Ada", ShownInClips: true},
		{AccountId: bob, Name: "Bob"},
		{AccountId: guest, Name: "guest_X7K2", Guest: true},
	}}

	shown, err := rpc_player_names.New(module).Shown(t.Context(), []casts.AccountID{account(t, ada), account(t, bob), account(t, guest)})
	require.NoError(t, err)

	assert.Equal(t, map[casts.AccountID]string{account(t, ada): "Ada"}, shown)
	assert.Equal(t, []string{ada, bob, guest}, module.asked)
}

func TestNoAccountAsksNothing(t *testing.T) {
	module := &player{}

	shown, err := rpc_player_names.New(module).Shown(t.Context(), nil)
	require.NoError(t, err)

	assert.Empty(t, shown)
	assert.Empty(t, module.asked)
}
