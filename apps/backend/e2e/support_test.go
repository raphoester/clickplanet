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
	supportv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/support/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/support/v1/supportv1connect"
)

func (p *gamer) support() supportv1connect.SupportServiceClient {
	return supportv1connect.NewSupportServiceClient(http.DefaultClient, p.stack.baseURL)
}

func (p *gamer) suggest(text string) error {
	p.t.Helper()

	req := connect.NewRequest(&supportv1.SuggestFeatureRequest{Text: text})
	p.send(req.Header())
	if _, err := p.support().SuggestFeature(p.t.Context(), req); err != nil {
		return fmt.Errorf("SuggestFeature failed: %w", err)
	}
	return nil
}

func (s gameStack) notes(t *testing.T) []*supportv1.Note {
	t.Helper()

	res, err := supportv1connect.NewAdminServiceClient(http.DefaultClient, s.adminURL).ListNotes(t.Context(),
		connect.NewRequest(&supportv1.ListNotesRequest{}))
	require.NoError(t, err)
	return res.Msg.GetNotes()
}

func TestAGuestSuggestsAFeatureAndTheOperatorReadsItUnderItsAccount(t *testing.T) {
	game := startGame(t)
	guest := game.newPlayer(t)

	require.NoError(t, guest.suggest("  a night mode\r\nfor the globe  "))

	notes := game.notes(t)
	require.Len(t, notes, 1)
	assert.Equal(t, supportv1.NoteKind_NOTE_KIND_SUGGESTION, notes[0].GetKind())
	assert.Equal(t, guest.account(), notes[0].GetAccountId())
	assert.Equal(t, "a night mode\nfor the globe", notes[0].GetText())
	assert.WithinDuration(t, time.Now(), notes[0].GetSubmittedAt().AsTime(), time.Minute)
}

func TestASuggestionNeedsAnAccountAndAText(t *testing.T) {
	game := startGame(t)
	guest := game.newPlayer(t)

	anonymous := connect.NewRequest(&supportv1.SuggestFeatureRequest{Text: "a night mode"})
	anonymous.Header().Set("X-Real-IP", callerIP)
	_, err := guest.support().SuggestFeature(t.Context(), anonymous)
	assert.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))

	assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(guest.suggest("   ")))
	assert.Empty(t, game.notes(t))
}

func TestAnAddressSendsThreeNotesAtOnceAndACallWithNoSessionSpendsNone(t *testing.T) {
	game := startGame(t)
	guest := game.newPlayer(t)

	for range 5 {
		anonymous := connect.NewRequest(&supportv1.SuggestFeatureRequest{Text: "a night mode"})
		anonymous.Header().Set("X-Real-IP", callerIP)
		_, err := guest.support().SuggestFeature(t.Context(), anonymous)
		require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
	}

	for n := range 3 {
		require.NoError(t, guest.suggest(fmt.Sprintf("idea %d", n)))
	}
	assert.Equal(t, connect.CodeResourceExhausted, connect.CodeOf(guest.suggest("one too many")))
	assert.Len(t, game.notes(t), 3)
}

func TestADeletedAccountLosesItsNotes(t *testing.T) {
	game := startGame(t)
	leaving, staying := game.newPlayer(t), game.newPlayer(t)
	require.NoError(t, leaving.suggest("a night mode"))
	require.NoError(t, staying.suggest("more flags"))

	deletion := connect.NewRequest(&authv1.DeleteAccountRequest{})
	leaving.send(deletion.Header())
	_, err := authv1connect.NewAuthServiceClient(http.DefaultClient, game.baseURL).DeleteAccount(t.Context(), deletion)
	require.NoError(t, err)

	var notes []*supportv1.Note
	require.Eventually(t, func() bool {
		notes = game.notes(t)
		return len(notes) == 1
	}, 5*time.Second, 20*time.Millisecond)
	assert.Equal(t, "more flags", notes[0].GetText())
}

func TestTheOperatorsListIsNotOnThePublicRouter(t *testing.T) {
	game := startGame(t)

	_, err := supportv1connect.NewAdminServiceClient(http.DefaultClient, game.baseURL).ListNotes(t.Context(),
		connect.NewRequest(&supportv1.ListNotesRequest{}))

	assert.Equal(t, connect.CodeUnimplemented, connect.CodeOf(err))
}
