package get_roster_handler_test

import (
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/get_roster_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/presence"
)

type stubUseCase []presence.Entry

func (s stubUseCase) Execute() []presence.Entry {
	return s
}

func TestTheRosterIsMappedInItsOrderAndMayBeCached(t *testing.T) {
	res, err := get_roster_handler.New(stubUseCase{
		{Key: "k1", Name: "Ada_L", Tag: "aaaaaa", Country: "fr"},
		{Name: "guest_Bob", Tag: "bbbbbb", Country: "de", Guest: true},
	}).GetRoster(t.Context(), connect.NewRequest(&playerv1.GetRosterRequest{}))

	require.NoError(t, err)
	require.Len(t, res.Msg.GetEntries(), 2)
	assert.Equal(t, []string{"k1", "Ada_L", "aaaaaa", "fr"},
		[]string{res.Msg.GetEntries()[0].GetKey(), res.Msg.GetEntries()[0].GetName(), res.Msg.GetEntries()[0].GetTag(), res.Msg.GetEntries()[0].GetCountryId()})
	assert.False(t, res.Msg.GetEntries()[0].GetGuest())
	assert.Equal(t, "guest_Bob", res.Msg.GetEntries()[1].GetName())
	assert.True(t, res.Msg.GetEntries()[1].GetGuest())
	assert.Equal(t, "public, max-age=5", res.Header().Get("Cache-Control"))
}

func TestNobodyPlayingIsAnEmptyRoster(t *testing.T) {
	res, err := get_roster_handler.New(stubUseCase{}).GetRoster(t.Context(), connect.NewRequest(&playerv1.GetRosterRequest{}))

	require.NoError(t, err)
	assert.Empty(t, res.Msg.GetEntries())
}
