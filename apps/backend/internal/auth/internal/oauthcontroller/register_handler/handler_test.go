package register_handler_test

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/connections"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/connections/usecases/register_client_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/oauthcontroller/register_handler"
)

type stub struct{}

func (stub) Execute(_ context.Context, in register_client_usecase.In) (register_client_usecase.Out, error) {
	if in.RedirectURIs[0] != "https://claude.ai/api/mcp/auth_callback" {
		return register_client_usecase.Out{}, fmt.Errorf("%w: elsewhere", connections.ErrInvalidRedirect)
	}
	return register_client_usecase.Out{ClientID: "the id", Client: connections.ClientSnapshot{
		Name: in.Name, RedirectURIs: in.RedirectURIs, IssuedAt: time.Unix(1760788800, 0),
	}}, nil
}

func register(t *testing.T, body string) (int, map[string]any) {
	t.Helper()
	recorder := httptest.NewRecorder()
	register_handler.New(stub{}, slog.New(slog.DiscardHandler)).
		ServeHTTP(recorder, httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/oauth/register", strings.NewReader(body)))

	var answer map[string]any
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &answer))
	return recorder.Code, answer
}

func TestAnAppIsAnsweredItsIdAndWhatItGetsWhateverElseItAskedFor(t *testing.T) {
	status, answer := register(t, `{"client_name":"Claude","redirect_uris":["https://claude.ai/api/mcp/auth_callback"],
		"token_endpoint_auth_method":"client_secret_post","grant_types":["authorization_code","refresh_token"]}`)

	assert.Equal(t, http.StatusCreated, status)
	assert.Equal(t, map[string]any{
		"client_id":                  "the id",
		"client_id_issued_at":        float64(1760788800),
		"client_name":                "Claude",
		"redirect_uris":              []any{"https://claude.ai/api/mcp/auth_callback"},
		"grant_types":                []any{"authorization_code", "refresh_token"},
		"response_types":             []any{"code"},
		"token_endpoint_auth_method": "none",
	}, answer)
}

func TestARegistrationIsRefusedInRFC7591sWords(t *testing.T) {
	status, answer := register(t, `{"client_name":"Evil","redirect_uris":["https://evil.example/cb"]}`)
	assert.Equal(t, http.StatusBadRequest, status)
	assert.Equal(t, "invalid_redirect_uri", answer["error"])

	status, answer = register(t, `not json`)
	assert.Equal(t, http.StatusBadRequest, status)
	assert.Equal(t, "invalid_client_metadata", answer["error"])
}
