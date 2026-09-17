package e2e_test

import (
	"fmt"
	"net/http"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1/playerv1connect"
)

func (p *gamer) announce(country string, guestName string) error {
	p.t.Helper()

	req := connect.NewRequest(&playerv1.AnnounceRequest{CountryId: country, GuestName: guestName})
	p.send(req.Header())
	if _, err := p.players().Announce(p.t.Context(), req); err != nil {
		return fmt.Errorf("Announce failed: %w", err)
	}
	return nil
}

func (s gameStack) roster(t *testing.T) *connect.Response[playerv1.GetRosterResponse] {
	t.Helper()

	res, err := playerv1connect.NewPlayerServiceClient(http.DefaultClient, s.baseURL).
		GetRoster(t.Context(), connect.NewRequest(&playerv1.GetRosterRequest{}))
	require.NoError(t, err)
	return res
}

func TestTheRosterListsPlayersThenGuestsWithTheChatsTag(t *testing.T) {
	game := startGame(t)
	ada := game.newPlayer(t)
	ada.link("google-ada")
	_, err := ada.setName("Ada_L")
	require.NoError(t, err)
	bob := game.newPlayer(t)
	nameless := game.newPlayer(t)

	require.NoError(t, bob.announce("de", "Bob"))
	require.NoError(t, nameless.announce("jp", ""))
	require.NoError(t, ada.announce("fr", "ignored"))

	message, err := bob.post("Bob")
	require.NoError(t, err)
	tag := message.GetAuthorTag()

	roster := game.roster(t)
	assert.Equal(t, "public, max-age=5", roster.Header().Get("Cache-Control"))
	lines := make([][]any, 0, len(roster.Msg.GetEntries()))
	for _, entry := range roster.Msg.GetEntries() {
		lines = append(lines, []any{entry.GetName(), entry.GetTag(), entry.GetCountryId(), entry.GetGuest()})
	}
	assert.Equal(t, [][]any{
		{"Ada_L", tag, "fr", false},
		{"guest_" + tag, tag, "jp", true},
		{"guest_Bob", tag, "de", true},
	}, lines, "the roster shows the tag the chat shows for the same address")
}

func TestAnAnnounceWithNoTokenIsUnauthenticatedAndTheRosterNeedsNone(t *testing.T) {
	game := startGame(t)

	_, err := playerv1connect.NewPlayerServiceClient(http.DefaultClient, game.baseURL).
		Announce(t.Context(), connect.NewRequest(&playerv1.AnnounceRequest{CountryId: "fr"}))

	assert.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
	assert.Empty(t, game.roster(t).Msg.GetEntries())
}

func TestAnAnnounceForACountryThatIsNotOneIsInvalidArgument(t *testing.T) {
	game := startGame(t)

	err := game.newPlayer(t).announce("atlantis", "Bob")

	assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}
