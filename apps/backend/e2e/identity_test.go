package e2e_test

import (
	"net/http"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1/authv1connect"
	chatv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/chat/v1"
	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1/planetv1connect"
)

// The same browser, holding the token ResumeSession answers rather than the one CreateSession did.
func (p *gamer) resumed() *gamer {
	p.t.Helper()

	req := connect.NewRequest(&authv1.ResumeSessionRequest{})
	req.Header().Set("X-Real-IP", callerIP)
	if p.cookie != "" {
		req.Header().Set("Cookie", p.cookie)
	}
	res, err := authv1connect.NewAuthServiceClient(http.DefaultClient, p.stack.baseURL).ResumeSession(p.t.Context(), req)
	require.NoError(p.t, err)

	return &gamer{t: p.t, stack: p.stack, cookie: p.cookie, token: res.Msg.GetToken()}
}

func (p *gamer) tryClick(tile uint32, country string) error {
	p.t.Helper()

	req := connect.NewRequest(&planetv1.ClickRequest{TileId: tile, CountryId: country})
	p.send(req.Header())
	_, err := planetv1connect.NewClickServiceClient(http.DefaultClient, p.stack.baseURL).Click(p.t.Context(), req)
	return err //nolint:wrapcheck // the test reads the code.
}

func TestTheCookieResumesATokenThatNamesThePlayerAndCannotAct(t *testing.T) {
	game := startGame(t)
	ada := game.newPlayer(t)
	ada.click(1, "fr")
	message, err := ada.post()
	require.NoError(t, err)
	_, err = ada.react(message.GetId(), chatv1.Reaction_REACTION_FIRE, true)
	require.NoError(t, err)

	reader := ada.resumed()

	require.NotEmpty(t, reader.token)
	assert.True(t, reader.history()[0].GetReactions()[0].GetMine(), "a read knows its reader with no Turnstile check")
	require.Eventually(t, func() bool { return reader.stats().GetTilesTaken() == 1 }, 5*time.Second, 20*time.Millisecond)

	assert.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(reader.tryClick(2, "fr")), "a click needs the check")
	_, err = reader.post()
	assert.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err), "so does a post")
	_, err = reader.react(message.GetId(), chatv1.Reaction_REACTION_SKULL, true)
	assert.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err), "and a reaction")
	assert.NoError(t, ada.tryClick(2, "fr"), "the click token still acts")
}

func TestABrowserWithNoSessionResumesNothing(t *testing.T) {
	game := startGame(t)

	assert.Empty(t, (&gamer{t: t, stack: game}).resumed().token)
	assert.Empty(t, (&gamer{t: t, stack: game, cookie: "cp_sid=made-up"}).resumed().token)
}
