package metadata_handler_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/oauthcontroller/metadata_handler"
)

func TestTheMetadataSaysWhereEachEndpointIsAndThatPKCEIsAllThereIs(t *testing.T) {
	recorder := httptest.NewRecorder()
	metadata_handler.New("https://api.clickplanet.lol").ServeHTTP(recorder, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil))

	var metadata map[string]any
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &metadata))
	assert.Equal(t, map[string]any{ //nolint:gosec // endpoints, not credentials.
		"issuer":                                         "https://api.clickplanet.lol",
		"authorization_endpoint":                         "https://api.clickplanet.lol/oauth/authorize",
		"token_endpoint":                                 "https://api.clickplanet.lol/oauth/token",
		"registration_endpoint":                          "https://api.clickplanet.lol/oauth/register",
		"response_types_supported":                       []any{"code"},
		"grant_types_supported":                          []any{"authorization_code", "refresh_token"},
		"code_challenge_methods_supported":               []any{"S256"},
		"token_endpoint_auth_methods_supported":          []any{"none"},
		"authorization_response_iss_parameter_supported": true,
	}, metadata)
	assert.Equal(t, "application/json", recorder.Header().Get("Content-Type"))
}
