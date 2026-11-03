package rest_discord_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/discord"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/discord/rest_discord"
)

type call struct {
	Method        string
	Path          string
	Authorization string
	Body          string
}

type fakeDiscord struct {
	mu      sync.Mutex
	calls   []call
	answers map[string]func(w http.ResponseWriter)
}

func (f *fakeDiscord) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	f.mu.Lock()
	f.calls = append(f.calls, call{Method: r.Method, Path: r.URL.Path, Authorization: r.Header.Get("Authorization"), Body: string(body)})
	answer, found := f.answers[r.Method+" "+r.URL.Path]
	f.mu.Unlock()
	if !found {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	answer(w)
}

func answering(status int, body string) func(w http.ResponseWriter) {
	return func(w http.ResponseWriter) {
		w.WriteHeader(status)
		_, _ = io.WriteString(w, body)
	}
}

func client(t *testing.T, answers map[string]func(w http.ResponseWriter)) (*rest_discord.Client, *fakeDiscord) {
	t.Helper()
	fake := &fakeDiscord{answers: answers}
	server := httptest.NewServer(fake)
	t.Cleanup(server.Close)
	return rest_discord.New(rest_discord.App{BotToken: "the-bot", Guild: "900"},
		rest_discord.Endpoints{API: server.URL + "/api"}, server.Client()), fake
}

func TestANewMemberJoinsWithItsRoles(t *testing.T) {
	c, fake := client(t, map[string]func(w http.ResponseWriter){
		"PUT /api/guilds/900/members/4242": answering(http.StatusCreated, `{}`),
	})

	require.NoError(t, c.Join(t.Context(), "4242", "the-token", []discord.RoleID{"1", "2"}))

	require.Len(t, fake.calls, 1)
	assert.Equal(t, "Bot the-bot", fake.calls[0].Authorization)
	assert.JSONEq(t, `{"access_token":"the-token","roles":["1","2"]}`, fake.calls[0].Body)
}

func TestAMemberAlreadyInTheServerIsGrantedEachRole(t *testing.T) {
	c, fake := client(t, nil)

	require.NoError(t, c.Join(t.Context(), "4242", "the-token", []discord.RoleID{"1", "2"}))

	paths := make([]string, 0, len(fake.calls))
	for _, made := range fake.calls {
		paths = append(paths, made.Method+" "+made.Path)
	}
	assert.Equal(t, []string{
		"PUT /api/guilds/900/members/4242",
		"PUT /api/guilds/900/members/4242/roles/1",
		"PUT /api/guilds/900/members/4242/roles/2",
	}, paths)
}

func TestAUserWhoLeftTheServerIsNoMember(t *testing.T) {
	c, _ := client(t, map[string]func(w http.ResponseWriter){
		"DELETE /api/guilds/900/members/4242/roles/1": answering(http.StatusNotFound, `{"code":10007,"message":"Unknown Member"}`),
		"PUT /api/guilds/900/members/4242/roles/2":    answering(http.StatusForbidden, `{"code":50013,"message":"Missing Permissions"}`),
	})

	assert.ErrorIs(t, c.Revoke(t.Context(), "4242", "1"), discord.ErrNotMember)
	assert.ErrorIs(t, c.Grant(t.Context(), "4242", "2"), discord.ErrRefused)
}

func TestARoleIsMadeMentionableInItsColor(t *testing.T) {
	c, fake := client(t, map[string]func(w http.ResponseWriter){
		"POST /api/guilds/900/roles": answering(http.StatusOK, `{"id":"77"}`),
	})

	role, err := c.MakeRole(t.Context(), "Ireland", 0x169b62)
	require.NoError(t, err)

	assert.Equal(t, discord.RoleID("77"), role)
	assert.JSONEq(t, `{"name":"Ireland","color":1481570,"mentionable":true}`, fake.calls[0].Body)
}

func TestARoleIsPainted(t *testing.T) {
	c, fake := client(t, nil)

	require.NoError(t, c.PaintRole(t.Context(), "77", 0x169b62))

	assert.Equal(t, "PATCH /api/guilds/900/roles/77", fake.calls[0].Method+" "+fake.calls[0].Path)
	assert.JSONEq(t, `{"color":1481570}`, fake.calls[0].Body)
}

func TestARoleDeletedByHandIsNoRoleToPaintAndDeletedAlready(t *testing.T) {
	c, _ := client(t, map[string]func(w http.ResponseWriter){
		"PATCH /api/guilds/900/roles/77":  answering(http.StatusNotFound, `{"code":10011,"message":"Unknown Role"}`),
		"DELETE /api/guilds/900/roles/77": answering(http.StatusNotFound, `{"code":10011,"message":"Unknown Role"}`),
	})

	assert.ErrorIs(t, c.PaintRole(t.Context(), "77", 0x169b62), discord.ErrNoRole)
	assert.NoError(t, c.DeleteRole(t.Context(), "77"))
}

func TestAPostPingsItsRoleAlone(t *testing.T) {
	c, fake := client(t, map[string]func(w http.ResponseWriter){
		"POST /api/channels/100/messages": answering(http.StatusOK, `{"id":"1"}`),
	})

	require.NoError(t, c.Post(t.Context(), "100", "<@&77> @everyone go", discord.RoleAudience("77")))

	var sent map[string]any
	require.NoError(t, json.Unmarshal([]byte(fake.calls[0].Body), &sent))
	assert.Equal(t, map[string]any{
		"content":          "<@&77> @everyone go",
		"allowed_mentions": map[string]any{"parse": []any{}, "roles": []any{"77"}},
	}, sent)
}

func TestAPostPingsItsRolesAloneAndNoRolePingsNobody(t *testing.T) {
	c, fake := client(t, map[string]func(w http.ResponseWriter){
		"POST /api/channels/100/messages": answering(http.StatusOK, `{"id":"1"}`),
	})

	require.NoError(t, c.Post(t.Context(), "100", "<@&77> <@&78> <@&79> go", discord.RoleAudience("77", "78")))
	require.NoError(t, c.Post(t.Context(), "100", "<@&77> @everyone go", discord.RoleAudience()))

	var several, none map[string]any
	require.NoError(t, json.Unmarshal([]byte(fake.calls[0].Body), &several))
	require.NoError(t, json.Unmarshal([]byte(fake.calls[1].Body), &none))
	assert.Equal(t, map[string]any{"parse": []any{}, "roles": []any{"77", "78"}}, several["allowed_mentions"])
	assert.Equal(t, map[string]any{"parse": []any{}, "roles": []any{}}, none["allowed_mentions"])
}

func TestAPostToEverybodyPingsEverybody(t *testing.T) {
	c, fake := client(t, map[string]func(w http.ResponseWriter){
		"POST /api/channels/100/messages": answering(http.StatusOK, `{"id":"1"}`),
	})

	require.NoError(t, c.Post(t.Context(), "100", "@everyone go", discord.Everybody()))

	var sent map[string]any
	require.NoError(t, json.Unmarshal([]byte(fake.calls[0].Body), &sent))
	assert.Equal(t, map[string]any{"parse": []any{"everyone"}}, sent["allowed_mentions"])
}

func TestABusyDiscordIsAFailureToTryAgain(t *testing.T) {
	c, _ := client(t, map[string]func(w http.ResponseWriter){
		"POST /api/channels/100/messages": answering(http.StatusTooManyRequests, `{"retry_after":1}`),
	})

	err := c.Post(t.Context(), "100", "go", discord.RoleAudience("77"))
	require.Error(t, err)
	assert.NotErrorIs(t, err, discord.ErrRefused)
}
