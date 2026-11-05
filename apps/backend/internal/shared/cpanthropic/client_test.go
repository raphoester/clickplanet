package cpanthropic_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpanthropic"
)

const key = "sk-ant-test-key"

func TestAClientWithNoKeyIsRefused(t *testing.T) {
	_, err := cpanthropic.NewClient(cpanthropic.Config{})

	require.ErrorIs(t, err, cpanthropic.ErrNoKey)
}

func refusing(t *testing.T, keys *[]string) *httptest.Server {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*keys = append(*keys, r.Header.Get("X-Api-Key"))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"type":"error","error":{"type":"invalid_request_error","message":"bad"}}`))
	}))
	t.Cleanup(server.Close)
	return server
}

func call(t *testing.T, client *anthropic.Client) error {
	t.Helper()

	_, err := client.Messages.New(t.Context(), anthropic.MessageNewParams{
		Model: "claude-haiku-5-5", MaxTokens: 10,
		Messages: []anthropic.MessageParam{anthropic.NewUserMessage(anthropic.NewTextBlock("hi"))},
	}, option.WithMaxRetries(0))
	return err //nolint:wrapcheck // the SDK's own error, as a caller first gets it: what Redacted is tested on.
}

func TestTheClientSendsItsKeyToItsBaseURL(t *testing.T) {
	var keys []string
	client, err := cpanthropic.NewClient(cpanthropic.Config{APIKey: key, BaseURL: refusing(t, &keys).URL})
	require.NoError(t, err)

	require.Error(t, call(t, client))

	assert.Equal(t, []string{key}, keys)
}

// The logs of a public repo are public: an error says what went wrong, never the request it carried.
func TestARedactedErrorSaysItsStatusAndNeverTheKey(t *testing.T) {
	var keys []string
	client, err := cpanthropic.NewClient(cpanthropic.Config{APIKey: key, BaseURL: refusing(t, &keys).URL})
	require.NoError(t, err)

	failed := call(t, client)
	redacted := cpanthropic.Redacted(failed)

	assert.Contains(t, failed.Error(), "POST", "the SDK's own error dumps the request")
	assert.Equal(t, "HTTP 400: invalid_request_error", redacted.Error())
	assert.NotContains(t, redacted.Error(), key)
}

func TestARedactedTimeoutStaysATimeout(t *testing.T) {
	assert.ErrorIs(t, cpanthropic.Redacted(context.DeadlineExceeded), context.DeadlineExceeded)
	assert.Equal(t, "no answer", cpanthropic.Redacted(errors.New("dial tcp 10.0.0.1: refused")).Error())
}
