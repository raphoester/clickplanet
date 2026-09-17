package e2e_test

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1/authv1connect"
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

// names is the roster as its names.
func (s gameStack) names(t *testing.T) []string {
	t.Helper()

	entries := s.roster(t).Msg.GetEntries()
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		names = append(names, entry.GetName())
	}
	return names
}

func (p *gamer) signOut() {
	p.t.Helper()

	req := connect.NewRequest(&authv1.SignOutRequest{})
	p.send(req.Header())
	_, err := authv1connect.NewAuthServiceClient(http.DefaultClient, p.stack.baseURL).SignOut(p.t.Context(), req)
	require.NoError(p.t, err)
}

func TestAGuestWhoSignsInAndPicksANameIsOneLineUnderItWithNoAnnounce(t *testing.T) {
	game := startGame(t)
	ada := game.newPlayer(t)
	require.NoError(t, ada.announce("fr", "Bob"))
	require.Equal(t, []string{"guest_Bob"}, game.names(t))

	ada.link("google-ada")
	_, err := ada.setName("Ada_L")
	require.NoError(t, err)

	assert.Equal(t, []string{"Ada_L"}, game.names(t), "the rename is on the roster as soon as SetName answers")
}

func TestAGuestWhoSignsInToAKnownAccountTakesItsNameAndLeavesNoGuestBehind(t *testing.T) {
	game := startGame(t)
	laptop := game.newPlayer(t)
	laptop.link("google-ada")
	_, err := laptop.setName("Ada_L")
	require.NoError(t, err)
	phone := game.newPlayer(t)
	require.NoError(t, phone.announce("fr", "Bob"))
	require.Equal(t, []string{"guest_Bob"}, game.names(t))

	phone.signIn("google-ada", authv1.SignInIntent_SIGN_IN_INTENT_SIGN_IN, authv1.SignInOutcome_SIGN_IN_OUTCOME_SIGNED_IN)

	assert.Eventually(t, func() bool {
		names := game.names(t)
		return len(names) == 1 && names[0] == "Ada_L"
	}, 5*time.Second, 20*time.Millisecond, "the event reaches the roster")
}

func TestASignedOutPlayerLeavesTheRosterAtOnce(t *testing.T) {
	game := startGame(t)
	ada := game.newPlayer(t)
	require.NoError(t, ada.announce("fr", "Bob"))
	require.Equal(t, []string{"guest_Bob"}, game.names(t))

	ada.signOut()

	assert.Eventually(t, func() bool { return len(game.names(t)) == 0 }, 5*time.Second, 20*time.Millisecond,
		"the event reaches the roster")
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

// rosterStream reads player.v1.PlayerService/ListenForEvents on a goroutine, until the test ends.
func (s gameStack) rosterStream(t *testing.T) <-chan *playerv1.PlayerEvent {
	t.Helper()

	stream, err := playerv1connect.NewPlayerServiceClient(http.DefaultClient, s.baseURL).
		ListenForEvents(t.Context(), connect.NewRequest(&playerv1.ListenForEventsRequest{}))
	require.NoError(t, err)

	events := make(chan *playerv1.PlayerEvent, 16)
	go func() {
		defer close(events)
		for stream.Receive() {
			events <- stream.Msg()
		}
	}()
	return events
}

func next(t *testing.T, events <-chan *playerv1.PlayerEvent) *playerv1.PlayerEvent {
	t.Helper()

	select {
	case event, open := <-events:
		require.True(t, open, "the stream ended")
		return event
	case <-time.After(5 * time.Second):
		require.FailNow(t, "no event")
		return nil
	}
}

func TestTheStreamSendsTheRosterThenEachJoinRenameAndLeave(t *testing.T) {
	game := startGame(t)
	bob := game.newPlayer(t)
	require.NoError(t, bob.announce("de", "Bob"))
	events := game.rosterStream(t)

	roster := next(t, events).GetRoster()
	require.NotNil(t, roster)
	require.Len(t, roster.GetEntries(), 1)
	assert.Equal(t, "guest_Bob", roster.GetEntries()[0].GetName())
	bobKey := roster.GetEntries()[0].GetKey()
	assert.NotEmpty(t, bobKey)

	ada := game.newPlayer(t)
	require.NoError(t, ada.announce("fr", "Ada"))
	joined := next(t, events).GetEntry()
	require.NotNil(t, joined)
	assert.Equal(t, "guest_Ada", joined.GetName())

	ada.link("google-ada")
	_, err := ada.setName("Ada_L")
	require.NoError(t, err)
	renamed := next(t, events).GetEntry()
	require.NotNil(t, renamed)
	assert.Equal(t, "Ada_L", renamed.GetName())
	assert.Equal(t, joined.GetKey(), renamed.GetKey(), "the same line, renamed")

	leave := connect.NewRequest(&playerv1.LeaveRequest{})
	bob.send(leave.Header())
	_, err = bob.players().Leave(t.Context(), leave)
	require.NoError(t, err)
	assert.Equal(t, bobKey, next(t, events).GetLeft().GetKey())
	assert.Equal(t, []string{"Ada_L"}, game.names(t))
}
