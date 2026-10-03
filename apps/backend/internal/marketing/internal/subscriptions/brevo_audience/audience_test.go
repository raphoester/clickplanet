package brevo_audience_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/marketing/internal/subscriptions/brevo_audience"
)

var config = brevo_audience.Config{APIKey: "the-key", ListID: 7, DOITemplateID: 12}

type seen struct {
	method string
	path   string
	query  string
	key    string
	body   map[string]any
}

func audience(t *testing.T, status int, answer string) (*brevo_audience.Audience, *seen) {
	t.Helper()

	got := &seen{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got.method, got.path, got.query, got.key = r.Method, r.URL.EscapedPath(), r.URL.RawQuery, r.Header.Get("api-key")
		raw, err := io.ReadAll(r.Body)
		assert.NoError(t, err)
		if len(raw) > 0 {
			assert.NoError(t, json.Unmarshal(raw, &got.body))
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = fmt.Fprint(w, answer)
	}))
	t.Cleanup(server.Close)

	a, err := brevo_audience.New(config, server.URL, server.Client())
	require.NoError(t, err)
	return a, got
}

func TestJoinPutsTheContactOnTheListAndLiftsAnyBlacklisting(t *testing.T) {
	a, got := audience(t, http.StatusCreated, `{"id":42}`)

	require.NoError(t, a.Join(t.Context(), "ada@example.com"))

	assert.Equal(t, http.MethodPost, got.method)
	assert.Equal(t, "/contacts", got.path)
	assert.Equal(t, "the-key", got.key)
	assert.Equal(t, map[string]any{
		"email": "ada@example.com", "listIds": []any{float64(7)}, "updateEnabled": true, "emailBlacklisted": false,
	}, got.body)
}

func TestAJoinThatUpdatesAnExistingContactIsASuccess(t *testing.T) {
	a, _ := audience(t, http.StatusNoContent, "")

	assert.NoError(t, a.Join(t.Context(), "ada@example.com"))
}

func TestInviteAsksBrevoForTheConfirmationEmail(t *testing.T) {
	a, got := audience(t, http.StatusCreated, "")

	require.NoError(t, a.Invite(t.Context(), "ada@example.com"))

	assert.Equal(t, http.MethodPost, got.method)
	assert.Equal(t, "/contacts/doubleOptinConfirmation", got.path)
	assert.Equal(t, map[string]any{
		"email": "ada@example.com", "includeListIds": []any{float64(7)}, "templateId": float64(12),
		"redirectionUrl": "https://clickplanet.lol/play",
	}, got.body)
}

func TestJoinedIsWhetherTheContactIsOnTheList(t *testing.T) {
	for name, c := range map[string]struct {
		answer string
		joined bool
	}{
		"on the list":       {`{"email":"ada@example.com","listIds":[3,7],"emailBlacklisted":false}`, true},
		"on another list":   {`{"email":"ada@example.com","listIds":[3],"emailBlacklisted":false}`, false},
		"on it, but out":    {`{"email":"ada@example.com","listIds":[7],"emailBlacklisted":true}`, false},
		"on no list at all": {`{"email":"ada@example.com","listIds":[]}`, false},
	} {
		t.Run(name, func(t *testing.T) {
			a, got := audience(t, http.StatusOK, c.answer)

			joined, err := a.Joined(t.Context(), "ada+season@example.com")

			require.NoError(t, err)
			assert.Equal(t, c.joined, joined)
			assert.Equal(t, http.MethodGet, got.method)
			assert.Equal(t, "/contacts/ada%2Bseason%40example.com", got.path)
			assert.Equal(t, "identifierType=email_id", got.query)
		})
	}
}

func TestAContactBrevoDoesNotKnowHasNotJoined(t *testing.T) {
	a, _ := audience(t, http.StatusNotFound, `{"code":"document_not_found","message":"Contact does not exist"}`)

	joined, err := a.Joined(t.Context(), "ada@example.com")

	require.NoError(t, err)
	assert.False(t, joined)
}

func TestLeaveBlacklistsTheContactAsBrevosOwnLinkDoes(t *testing.T) {
	a, got := audience(t, http.StatusNoContent, "")

	require.NoError(t, a.Leave(t.Context(), "ada@example.com"))

	assert.Equal(t, http.MethodPut, got.method)
	assert.Equal(t, "/contacts/ada%40example.com", got.path)
	assert.Equal(t, map[string]any{"emailBlacklisted": true}, got.body)
}

func TestForgetDeletesTheContact(t *testing.T) {
	a, got := audience(t, http.StatusNoContent, "")

	require.NoError(t, a.Forget(t.Context(), "ada@example.com"))

	assert.Equal(t, http.MethodDelete, got.method)
	assert.Equal(t, "/contacts/ada%40example.com", got.path)
}

func TestLeavingOrForgettingAContactBrevoDoesNotKnowIsASuccess(t *testing.T) {
	a, _ := audience(t, http.StatusNotFound, `{"code":"document_not_found","message":"Contact does not exist"}`)

	require.NoError(t, a.Leave(t.Context(), "ada@example.com"))
	assert.NoError(t, a.Forget(t.Context(), "ada@example.com"))
}

func TestARefusalIsAnErrorThatSaysWhy(t *testing.T) {
	a, _ := audience(t, http.StatusUnauthorized, `{"code":"unauthorized","message":"Key not found"}`)

	err := a.Join(t.Context(), "ada@example.com")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "401")
	assert.Contains(t, err.Error(), "unauthorized Key not found")
	assert.NotContains(t, err.Error(), "ada")
}

func TestAnAnswerThatIsNotJSONIsAnError(t *testing.T) {
	a, _ := audience(t, http.StatusOK, "<html>bad gateway</html>")

	_, err := a.Joined(t.Context(), "ada@example.com")

	assert.Error(t, err)
}

func TestAnUnreachableBrevoIsAnError(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	server.Close()
	a, err := brevo_audience.New(config, server.URL, http.DefaultClient)
	require.NoError(t, err)

	err = a.Leave(t.Context(), "ada@example.com")

	require.Error(t, err)
	assert.NotContains(t, err.Error(), "ada", "an error is logged, and the address stays out of the log")
}

func TestNewRefusesAMissingKeyListOrTemplate(t *testing.T) {
	for name, c := range map[string]brevo_audience.Config{
		"no key":      {ListID: 7, DOITemplateID: 12},
		"no list":     {APIKey: "k", DOITemplateID: 12},
		"no template": {APIKey: "k", ListID: 7},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := brevo_audience.New(c, brevo_audience.Production, http.DefaultClient)

			assert.Error(t, err)
		})
	}
}

func TestTheConfigNeverPrintsTheKey(t *testing.T) {
	assert.NotContains(t, fmt.Sprintf("%v %+v", config, struct{ Brevo brevo_audience.Config }{config}), "the-key")
}
