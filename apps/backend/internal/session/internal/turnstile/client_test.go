package turnstile

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	client, err := New(Config{
		Secret:    "a-secret",
		Hostnames: []string{"clickplanet.lol"},
		Action:    "session",
	})
	require.NoError(t, err)

	client.endpoint = server.URL
	return client
}

func answering(body string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, body)
	}
}

func TestNewRefusesAConfigThatWouldRefuseEveryToken(t *testing.T) {
	valid := Config{Secret: "s", Hostnames: []string{"clickplanet.lol"}, Action: "session"}

	_, err := New(valid)
	assert.NoError(t, err)

	noSecret := valid
	noSecret.Secret = ""
	_, err = New(noSecret)
	assert.Error(t, err)

	noHostnames := valid
	noHostnames.Hostnames = nil
	_, err = New(noHostnames)
	assert.Error(t, err)

	noAction := valid
	noAction.Action = ""
	_, err = New(noAction)
	assert.Error(t, err)
}

func TestAGoodTokenIsAccepted(t *testing.T) {
	client := newClient(t, answering(`{"success":true,"action":"session","hostname":"clickplanet.lol"}`))

	assert.NoError(t, client.Verify(context.Background(), "a-token", "203.0.113.7"))
}

func TestTheSecretAndTheTokenAreSentAsAForm(t *testing.T) {
	var got url.Values

	client := newClient(t, func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, r.ParseForm())
		got = r.PostForm
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"success":true,"action":"session","hostname":"clickplanet.lol"}`)
	})

	require.NoError(t, client.Verify(context.Background(), "a-token", "203.0.113.7"))

	assert.Equal(t, "a-secret", got.Get("secret"))
	assert.Equal(t, "a-token", got.Get("response"))
	assert.Equal(t, "203.0.113.7", got.Get("remoteip"))
}

func TestARefusedTokenIsRefused(t *testing.T) {
	client := newClient(t, answering(`{"success":false,"error-codes":["invalid-input-response"]}`))

	assert.ErrorIs(t, client.Verify(context.Background(), "a-token", ""), ErrRefused)
}

// The sitekey is public, so a token can be minted from anywhere the widget is
// embedded. The action and the hostname are what tie it back to this surface.
func TestATokenForAnotherActionOrHostnameIsRefused(t *testing.T) {
	otherAction := newClient(t, answering(`{"success":true,"action":"signup","hostname":"clickplanet.lol"}`))
	assert.ErrorIs(t, otherAction.Verify(context.Background(), "a-token", ""), ErrRefused)

	otherHost := newClient(t, answering(`{"success":true,"action":"session","hostname":"evil.example"}`))
	assert.ErrorIs(t, otherHost.Verify(context.Background(), "a-token", ""), ErrRefused)
}

func TestAnEmptyOrOversizedTokenIsRefusedWithoutCallingSiteverify(t *testing.T) {
	called := false
	client := newClient(t, func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"success":true,"action":"session","hostname":"clickplanet.lol"}`)
	})

	assert.ErrorIs(t, client.Verify(context.Background(), "", ""), ErrRefused)
	assert.ErrorIs(t, client.Verify(context.Background(), strings.Repeat("a", maxTokenLength+1), ""), ErrRefused)
	assert.False(t, called)
}

// Failing open would make the check decorative: an attacker who can reach the
// backend can also make siteverify unreachable from it.
func TestEveryUpstreamFailureIsARefusal(t *testing.T) {
	t.Run("non-2xx", func(t *testing.T) {
		client := newClient(t, func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		})
		assert.ErrorIs(t, client.Verify(context.Background(), "a-token", ""), ErrRefused)
	})

	t.Run("body that is not JSON", func(t *testing.T) {
		client := newClient(t, answering(`<html>down for maintenance</html>`))
		assert.ErrorIs(t, client.Verify(context.Background(), "a-token", ""), ErrRefused)
	})

	t.Run("unreachable", func(t *testing.T) {
		client := newClient(t, answering(`{"success":true}`))
		client.endpoint = "http://127.0.0.1:1/nothing-here"
		assert.ErrorIs(t, client.Verify(context.Background(), "a-token", ""), ErrRefused)
	})

	t.Run("times out", func(t *testing.T) {
		client := newClient(t, func(w http.ResponseWriter, _ *http.Request) {
			time.Sleep(200 * time.Millisecond)
			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprint(w, `{"success":true,"action":"session","hostname":"clickplanet.lol"}`)
		})
		client.http.Timeout = 10 * time.Millisecond

		assert.ErrorIs(t, client.Verify(context.Background(), "a-token", ""), ErrRefused)
	})
}

// The secret must never reach a log or an error a caller can read.
func TestTheSecretIsNotInTheRefusal(t *testing.T) {
	client := newClient(t, answering(`{"success":false,"error-codes":["invalid-input-secret"]}`))

	err := client.Verify(context.Background(), "a-token", "")
	require.Error(t, err)
	assert.NotContains(t, err.Error(), "a-secret")
}
