package cloudflare_mailer_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/signin"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/signin/cloudflare_mailer"
)

var (
	config = cloudflare_mailer.Config{AccountID: "the-account", APIToken: "the-token"}
	sender = cloudflare_mailer.Sender{Address: "noreply@example.com", Name: "ClickPlanet"}
	letter = signin.Letter{Subject: "the subject", Text: "the text", HTML: "<p>the html</p>"}
)

const delivered = `{"success":true,"errors":[],"messages":[],"result":{"delivered":["player@example.com"],"permanent_bounces":[],"queued":[]}}`

func mailer(t *testing.T, handler http.HandlerFunc) *cloudflare_mailer.Mailer {
	t.Helper()

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	m, err := cloudflare_mailer.New(config, sender, server.URL, server.Client())
	require.NoError(t, err)
	return m
}

func answering(status int, body string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = fmt.Fprint(w, body)
	}
}

func TestALetterIsPostedToTheAccountWithTheToken(t *testing.T) {
	var got *http.Request
	var sent map[string]any
	m := mailer(t, func(w http.ResponseWriter, r *http.Request) {
		got = r
		raw, err := io.ReadAll(r.Body)
		assert.NoError(t, err)
		assert.NoError(t, json.Unmarshal(raw, &sent))
		answering(http.StatusOK, delivered)(w, r)
	})

	require.NoError(t, m.Send(t.Context(), "player@example.com", letter))

	assert.Equal(t, http.MethodPost, got.Method)
	assert.Equal(t, "/accounts/the-account/email/sending/send", got.URL.Path)
	assert.Equal(t, "Bearer the-token", got.Header.Get("Authorization"))
	assert.Equal(t, map[string]any{
		"to":      "player@example.com",
		"from":    map[string]any{"address": "noreply@example.com", "name": "ClickPlanet"},
		"subject": "the subject",
		"text":    "the text",
		"html":    "<p>the html</p>",
	}, sent)
}

func TestARefusalIsAnErrorThatSaysWhy(t *testing.T) {
	m := mailer(t, answering(http.StatusForbidden,
		`{"success":false,"errors":[{"code":10105,"message":"email.sending.error.authentication.not_entitled"}],"messages":[],"result":null}`))

	err := m.Send(t.Context(), "player@example.com", letter)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "403")
	assert.Contains(t, err.Error(), "10105 email.sending.error.authentication.not_entitled")
}

func TestAPermanentBounceIsAnError(t *testing.T) {
	m := mailer(t, answering(http.StatusOK,
		`{"success":true,"errors":[],"messages":[],"result":{"delivered":[],"permanent_bounces":["player@example.com"],"queued":[]}}`))

	assert.Error(t, m.Send(t.Context(), "player@example.com", letter))
}

func TestAQueuedLetterIsSent(t *testing.T) {
	m := mailer(t, answering(http.StatusOK,
		`{"success":true,"errors":[],"messages":[],"result":{"delivered":[],"permanent_bounces":[],"queued":["player@example.com"]}}`))

	assert.NoError(t, m.Send(t.Context(), "player@example.com", letter))
}

func TestABodyThatIsNotJSONIsAnError(t *testing.T) {
	m := mailer(t, answering(http.StatusBadGateway, "<html>bad gateway</html>"))

	assert.Error(t, m.Send(t.Context(), "player@example.com", letter))
}

func TestAnUnreachableAPIIsAnError(t *testing.T) {
	server := httptest.NewServer(answering(http.StatusOK, delivered))
	server.Close()
	m, err := cloudflare_mailer.New(config, sender, server.URL, http.DefaultClient)
	require.NoError(t, err)

	assert.Error(t, m.Send(t.Context(), "player@example.com", letter))
}

func TestNewRefusesAMissingCredentialOrSender(t *testing.T) {
	for name, c := range map[string]struct {
		config cloudflare_mailer.Config
		sender cloudflare_mailer.Sender
	}{
		"no account": {cloudflare_mailer.Config{APIToken: "t"}, sender},
		"no token":   {cloudflare_mailer.Config{AccountID: "a"}, sender},
		"no sender":  {config, cloudflare_mailer.Sender{}},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := cloudflare_mailer.New(c.config, c.sender, cloudflare_mailer.Production, http.DefaultClient)

			assert.Error(t, err)
		})
	}
}
