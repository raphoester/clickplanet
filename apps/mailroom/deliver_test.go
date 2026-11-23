package main

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mail struct {
	raw      string
	rejected []string
}

func (m *mail) To() string { return "support+1@clickplanet.lol" }

func (m *mail) Raw() io.ReadCloser { return io.NopCloser(bytes.NewBufferString(m.raw)) }

func (m *mail) SetReject(reason string) error {
	m.rejected = append(m.rejected, reason)
	return nil
}

func backend(t *testing.T, status int, seen *http.Request, body *[]byte) Backend {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*seen = *r.Clone(r.Context())
		read, err := io.ReadAll(r.Body)
		assert.NoError(t, err)
		*body = read
		w.WriteHeader(status)
	}))
	t.Cleanup(server.Close)
	return Backend{URL: server.URL + "/mailroom/replies", Secret: "the-secret", HTTP: server.Client()}
}

func TestTheRawMailAndItsRecipientArePostedWithTheSecret(t *testing.T) {
	var seen http.Request
	var body []byte
	sent := &mail{raw: "Message-ID: <1@x>\r\n\r\nHi"}

	require.NoError(t, backend(t, http.StatusAccepted, &seen, &body).Deliver(t.Context(), sent))

	assert.Equal(t, http.MethodPost, seen.Method)
	assert.Equal(t, "/mailroom/replies", seen.URL.Path)
	assert.Equal(t, "Bearer the-secret", seen.Header.Get("Authorization"))
	assert.Equal(t, "message/rfc822", seen.Header.Get("Content-Type"))
	assert.Equal(t, "support+1@clickplanet.lol", seen.Header.Get("X-Envelope-To"))
	assert.Equal(t, "Message-ID: <1@x>\r\n\r\nHi", string(body))
	assert.Empty(t, sent.rejected)
}

func TestAMailTheBackendDidNotKeepFailsSoItComesAgain(t *testing.T) {
	for _, status := range []int{http.StatusUnauthorized, http.StatusInternalServerError, http.StatusBadGateway} {
		var seen http.Request
		var body []byte

		err := backend(t, status, &seen, &body).Deliver(t.Context(), &mail{raw: "Hi"})

		require.Error(t, err, status)
	}
}

func TestAnUnreachableBackendFailsSoTheMailComesAgain(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	server.Close()

	err := Backend{URL: server.URL, Secret: "the-secret", HTTP: http.DefaultClient}.Deliver(t.Context(), &mail{raw: "Hi"})

	require.Error(t, err)
}

func TestAMailTheBackendFindsTooLargeIsRejectedForGood(t *testing.T) {
	var seen http.Request
	var body []byte
	sent := &mail{raw: "Hi"}

	require.NoError(t, backend(t, http.StatusRequestEntityTooLarge, &seen, &body).Deliver(t.Context(), sent))

	assert.Equal(t, []string{"This mail is too large."}, sent.rejected)
}
