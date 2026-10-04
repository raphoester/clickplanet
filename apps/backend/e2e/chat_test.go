package e2e_test

import (
	"fmt"
	"net/http"
	"regexp"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1/authv1connect"
	chatv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/chat/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/chat/v1/chatv1connect"
	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpconnect"
)

var (
	guestName     = regexp.MustCompile(`^guest_[0-9a-f]{6}$`)
	generatedName = regexp.MustCompile(`^[A-Z][a-z]+[A-Z][a-z]+[0-9]{2}$`)
)

func (p *gamer) post() (*chatv1.ChatMessage, error) {
	p.t.Helper()

	req := connect.NewRequest(&chatv1.SendMessageRequest{AuthorId: "browser-1", CountryId: "fr", Text: "hello"})
	p.send(req.Header())
	res, err := chatv1connect.NewChatServiceClient(http.DefaultClient, p.stack.baseURL).SendMessage(p.t.Context(), req)
	if err != nil {
		return nil, fmt.Errorf("SendMessage failed: %w", err)
	}
	return res.Msg.GetMessage(), nil
}

func TestAPlayerWithAUsernamePostsUnderIt(t *testing.T) {
	game := startGame(t)
	ada := game.newPlayer(t)
	ada.link("google-ada")
	_, err := ada.setName("Ada_L")
	require.NoError(t, err)

	message, err := ada.post()

	require.NoError(t, err)
	assert.Equal(t, "Ada_L", message.GetAuthorName())
}

func TestAPlayerPostsInTheColorItChoseAndWithItsStreak(t *testing.T) {
	game := startGame(t)
	ada := game.newPlayer(t)
	ada.link("google-ada")
	_, err := ada.setName("Ada_L")
	require.NoError(t, err)
	require.NoError(t, ada.setColor(playerv1.NameColor_NAME_COLOR_TEAL))
	ada.click(1, "fr")
	require.Eventually(t, func() bool { return ada.stats().GetStreakCurrent() == 1 }, 5*time.Second, 20*time.Millisecond)

	message, err := ada.post()

	require.NoError(t, err)
	assert.Equal(t, playerv1.NameColor_NAME_COLOR_TEAL, message.GetAuthorColor())
	assert.Equal(t, uint32(1), message.GetAuthorStreak())
}

func TestEachMessageSentClimbsTheChatterTrackAndTakesNoTile(t *testing.T) {
	game := startGame(t)
	ada := game.newPlayer(t)
	ada.link("google-ada")

	for range 3 {
		_, err := ada.post()
		require.NoError(t, err)
	}

	chatter := func() *playerv1.Track {
		req := connect.NewRequest(&playerv1.GetTitlesRequest{})
		ada.send(req.Header())
		res, err := ada.players().GetTitles(t.Context(), req)
		require.NoError(t, err)
		for _, track := range res.Msg.GetTracks() {
			if track.GetId() == "chatter" {
				return track
			}
		}
		return nil
	}
	require.Eventually(t, func() bool { return chatter().GetProgress() == 3 }, 5*time.Second, 20*time.Millisecond,
		"chat publishes each message and player counts it")
	assert.Zero(t, ada.stats().GetTilesTaken())
	assert.Zero(t, ada.stats().GetStreakCurrent(), "a message is no take")
}

func TestAGuestMayNotChooseAColor(t *testing.T) {
	game := startGame(t)

	err := game.newPlayer(t).setColor(playerv1.NameColor_NAME_COLOR_TEAL)

	assert.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))
}

func TestAPlayerWithNoUsernamePostsUnderItsGuestCodeEveryTime(t *testing.T) {
	game := startGame(t)
	bob := game.newPlayer(t)

	first, err := bob.post()
	require.NoError(t, err)
	second, err := bob.post()
	require.NoError(t, err)

	assert.Regexp(t, guestName, first.GetAuthorName())
	assert.Equal(t, first.GetAuthorName(), second.GetAuthorName(), "the code is drawn once and kept")
	other, err := game.newPlayer(t).post()
	require.NoError(t, err)
	assert.NotEqual(t, first.GetAuthorName(), other.GetAuthorName(), "each account has its own code")
}

func TestASenderWithNoTokenIsUnauthenticated(t *testing.T) {
	game := startGame(t)
	nobody := &gamer{t: t, stack: game}

	_, err := nobody.post()

	assert.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
}

func (p *gamer) react(messageID string, reaction chatv1.Reaction, on bool) ([]*chatv1.ReactionCount, error) {
	p.t.Helper()

	req := connect.NewRequest(&chatv1.ReactRequest{MessageId: messageID, Reaction: reaction, On: on})
	p.send(req.Header())
	res, err := chatv1connect.NewChatServiceClient(http.DefaultClient, p.stack.baseURL).React(p.t.Context(), req)
	if err != nil {
		return nil, fmt.Errorf("React failed: %w", err)
	}
	return res.Msg.GetReactions(), nil
}

func (p *gamer) history() []*chatv1.ChatMessage {
	p.t.Helper()

	req := connect.NewRequest(&chatv1.GetHistoryRequest{})
	p.send(req.Header())
	res, err := chatv1connect.NewChatServiceClient(http.DefaultClient, p.stack.baseURL).GetHistory(p.t.Context(), req)
	require.NoError(p.t, err)
	return res.Msg.GetMessages()
}

func TestEachAccountIsItsOwnReactorAndNoAccountIsRefused(t *testing.T) {
	game := startGame(t)
	ada := game.newPlayer(t)
	ada.link("google-ada")
	_, err := ada.setName("Ada_L")
	require.NoError(t, err)
	bob := game.newPlayer(t)
	message, err := bob.post()
	require.NoError(t, err)

	counts, err := ada.react(message.GetId(), chatv1.Reaction_REACTION_CLOWN, true)
	require.NoError(t, err)
	assert.True(t, counts[0].GetMine())

	counts, err = bob.react(message.GetId(), chatv1.Reaction_REACTION_CLOWN, true)
	require.NoError(t, err)
	require.Len(t, counts, 1)
	assert.Equal(t, uint32(2), counts[0].GetCount(), "two accounts on one address are two reactors")

	counts, err = ada.react(message.GetId(), chatv1.Reaction_REACTION_CLOWN, false)
	require.NoError(t, err)
	assert.Equal(t, uint32(1), counts[0].GetCount())
	assert.False(t, counts[0].GetMine())

	reactions := bob.history()[0].GetReactions()
	require.Len(t, reactions, 1)
	assert.Equal(t, chatv1.Reaction_REACTION_CLOWN, reactions[0].GetReaction())
	assert.True(t, reactions[0].GetMine(), "the history says which reactions are the caller's")

	nobody := &gamer{t: t, stack: game}
	_, err = nobody.react(message.GetId(), chatv1.Reaction_REACTION_CLOWN, true)
	assert.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
}

func TestAReactionToNoMessageIsNotFound(t *testing.T) {
	game := startGame(t)

	_, err := game.newPlayer(t).react("no-such-message", chatv1.Reaction_REACTION_SKULL, true)

	assert.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
}

type credentials int

const (
	cookieOnly credentials = iota
	tokenOnly
)

func (p *gamer) sendOnly(header http.Header, with credentials) {
	header.Set("X-Real-IP", callerIP)
	if with == cookieOnly {
		header.Set("Cookie", p.cookie)
		return
	}
	header.Set(cpconnect.SessionHeader, p.token)
}

func (p *gamer) historyWith(with credentials) *chatv1.GetHistoryResponse {
	p.t.Helper()

	req := connect.NewRequest(&chatv1.GetHistoryRequest{})
	p.sendOnly(req.Header(), with)
	res, err := chatv1connect.NewChatServiceClient(http.DefaultClient, p.stack.baseURL).GetHistory(p.t.Context(), req)
	require.NoError(p.t, err)
	return res.Msg
}

func TestTheCookieAloneNamesWhoReadsTheHistory(t *testing.T) {
	game := startGame(t)
	ada := game.newPlayer(t)
	message, err := ada.post()
	require.NoError(t, err)
	_, err = ada.react(message.GetId(), chatv1.Reaction_REACTION_FIRE, true)
	require.NoError(t, err)

	reactions := ada.historyWith(cookieOnly).GetMessages()[0].GetReactions()

	require.Len(t, reactions, 1)
	assert.True(t, reactions[0].GetMine(), "a click token lasts an hour, the cookie months")
	nobody := &gamer{t: t, stack: game}
	assert.False(t, nobody.historyWith(cookieOnly).GetMessages()[0].GetReactions()[0].GetMine())
}

func (p *gamer) markSeen(at time.Time, with credentials) error {
	p.t.Helper()

	req := connect.NewRequest(&chatv1.MarkSeenRequest{SeenUntilUnixMs: at.UnixMilli()})
	p.sendOnly(req.Header(), with)
	_, err := chatv1connect.NewChatServiceClient(http.DefaultClient, p.stack.baseURL).MarkSeen(p.t.Context(), req)
	if err != nil {
		return fmt.Errorf("MarkSeen failed: %w", err)
	}
	return nil
}

func (p *gamer) seenUntil(with credentials) int64 {
	p.t.Helper()

	return p.historyWith(with).GetSeenUntilUnixMs()
}

func TestTheCookieAloneKeepsWhenThePlayerLastSawTheChat(t *testing.T) {
	game := startGame(t)
	ada := game.newPlayer(t)
	earlier := time.Now().Add(-time.Hour).Truncate(time.Millisecond)

	require.Zero(t, ada.seenUntil(cookieOnly), "nothing seen yet")
	require.NoError(t, ada.markSeen(earlier, cookieOnly))

	assert.Equal(t, earlier.UnixMilli(), ada.seenUntil(cookieOnly), "the click token lasts an hour, the cookie months")
	assert.Equal(t, earlier.UnixMilli(), ada.seenUntil(tokenOnly), "one account, either way in")
	assert.Zero(t, game.newPlayer(t).seenUntil(cookieOnly), "each account has its own mark")

	require.NoError(t, ada.markSeen(earlier.Add(-time.Minute), cookieOnly))
	assert.Equal(t, earlier.UnixMilli(), ada.seenUntil(cookieOnly), "a mark only moves forward")
}

func TestNoCookieAndNoTokenHasSeenNothingAndCannotMarkIt(t *testing.T) {
	game := startGame(t)
	nobody := &gamer{t: t, stack: game}

	assert.Zero(t, nobody.seenUntil(cookieOnly))
	assert.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(nobody.markSeen(time.Now(), cookieOnly)))
}

func TestADeletedAccountLosesItsSeenMark(t *testing.T) {
	game := startGame(t)
	ada := game.newPlayer(t)
	require.NoError(t, ada.markSeen(time.Now().Add(-time.Minute), cookieOnly))

	deletion := connect.NewRequest(&authv1.DeleteAccountRequest{})
	ada.send(deletion.Header())
	_, err := authv1connect.NewAuthServiceClient(http.DefaultClient, game.baseURL).DeleteAccount(t.Context(), deletion)
	require.NoError(t, err)

	require.Eventually(t, func() bool { return ada.seenUntil(tokenOnly) == 0 }, 5*time.Second, 20*time.Millisecond,
		"the token still names the deleted account, and chat forgot it")
}
