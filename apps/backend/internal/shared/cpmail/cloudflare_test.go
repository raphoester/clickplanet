package cpmail_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpmail"
)

var (
	account = cpmail.CloudflareConfig{AccountID: "the-account", APIToken: "the-token"}
	mail    = cpmail.Mail{
		From: cpmail.Named{Address: "support@example.com", Name: "ClickPlanet"}, To: "player@example.com",
		Subject: "Re: your idea", Text: "Thanks!", HTML: "<p>Thanks!</p>",
	}
)

const delivered = `{"success":true,"errors":[],"messages":[],"result":{"message_id":"<a1b2@example.com>",` +
	`"delivered":["player@example.com"],"permanent_bounces":[],"queued":[]}}`

func cloudflare(t *testing.T, handler http.HandlerFunc) *cpmail.Cloudflare {
	t.Helper()

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	m, err := cpmail.NewCloudflare(account, server.URL, server.Client())
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

func TestAMailIsPostedToTheAccountWithTheTokenAndItsIDComesBack(t *testing.T) {
	var got *http.Request
	var sent map[string]any
	m := cloudflare(t, func(w http.ResponseWriter, r *http.Request) {
		got = r
		raw, err := io.ReadAll(r.Body)
		assert.NoError(t, err)
		assert.NoError(t, json.Unmarshal(raw, &sent))
		answering(http.StatusOK, delivered)(w, r)
	})
	threaded := mail
	threaded.ReplyTo = "support+1@example.com"
	threaded.Headers = map[string]string{"In-Reply-To": "<x@example.org>"}

	id, err := m.Send(t.Context(), threaded)

	require.NoError(t, err)
	assert.Equal(t, cpmail.MessageID("a1b2@example.com"), id)
	assert.Equal(t, http.MethodPost, got.Method)
	assert.Equal(t, "/accounts/the-account/email/sending/send", got.URL.Path)
	assert.Equal(t, "Bearer the-token", got.Header.Get("Authorization"))
	assert.Equal(t, map[string]any{
		"to":       "player@example.com",
		"from":     map[string]any{"address": "support@example.com", "name": "ClickPlanet"},
		"reply_to": "support+1@example.com",
		"subject":  "Re: your idea",
		"text":     "Thanks!",
		"html":     "<p>Thanks!</p>",
		"headers":  map[string]any{"In-Reply-To": "<x@example.org>"},
	}, sent)
}

func TestARefusalIsAnErrorThatSaysWhy(t *testing.T) {
	m := cloudflare(t, answering(http.StatusForbidden,
		`{"success":false,"errors":[{"code":10105,"message":"email.sending.error.authentication.not_entitled"}],"messages":[],"result":null}`))

	_, err := m.Send(t.Context(), mail)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "403")
	assert.Contains(t, err.Error(), "10105 email.sending.error.authentication.not_entitled")
}

func TestAPermanentBounceIsAnError(t *testing.T) {
	m := cloudflare(t, answering(http.StatusOK,
		`{"success":true,"errors":[],"messages":[],"result":{"message_id":"<a@b>","delivered":[],"permanent_bounces":["player@example.com"],"queued":[]}}`))

	_, err := m.Send(t.Context(), mail)
	assert.Error(t, err)
}

func TestAQueuedMailIsSent(t *testing.T) {
	m := cloudflare(t, answering(http.StatusOK,
		`{"success":true,"errors":[],"messages":[],"result":{"message_id":"<a@b>","delivered":[],"permanent_bounces":[],"queued":["player@example.com"]}}`))

	_, err := m.Send(t.Context(), mail)
	assert.NoError(t, err)
}

func TestABodyThatIsNotJSONOrHasNoMessageIDIsAnError(t *testing.T) {
	for name, handler := range map[string]http.HandlerFunc{
		"not JSON":      answering(http.StatusBadGateway, "<html>bad gateway</html>"),
		"no message id": answering(http.StatusOK, `{"success":true,"errors":[],"messages":[],"result":{"delivered":["player@example.com"]}}`),
		"no result":     answering(http.StatusOK, `{"success":true,"errors":[],"messages":[]}`),
	} {
		_, err := cloudflare(t, handler).Send(t.Context(), mail)
		assert.Error(t, err, name)
	}
}

func TestAnUnreachableAPIIsAnError(t *testing.T) {
	server := httptest.NewServer(answering(http.StatusOK, delivered))
	server.Close()
	m, err := cpmail.NewCloudflare(account, server.URL, http.DefaultClient)
	require.NoError(t, err)

	_, err = m.Send(t.Context(), mail)
	assert.Error(t, err)
}

func TestNewRefusesAMissingCredential(t *testing.T) {
	for name, config := range map[string]cpmail.CloudflareConfig{
		"no account": {APIToken: "t"},
		"no token":   {AccountID: "a"},
	} {
		_, err := cpmail.NewCloudflare(config, cpmail.CloudflareAPI, http.DefaultClient)

		assert.Error(t, err, name)
	}
}
