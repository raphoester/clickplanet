package metadata_handler

import (
	"encoding/json"
	"net/http"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/oauthcontroller"
)

type metadata struct {
	Issuer                            string   `json:"issuer"`
	AuthorizationEndpoint             string   `json:"authorization_endpoint"`
	TokenEndpoint                     string   `json:"token_endpoint"`
	RegistrationEndpoint              string   `json:"registration_endpoint"`
	ResponseTypes                     []string `json:"response_types_supported"`
	GrantTypes                        []string `json:"grant_types_supported"`
	CodeChallengeMethods              []string `json:"code_challenge_methods_supported"`
	TokenEndpointAuthMethods          []string `json:"token_endpoint_auth_methods_supported"`
	AuthorizationResponseIssParameter bool     `json:"authorization_response_iss_parameter_supported"`
}

// RFC 8414: where an app finds the rest. Every app is a public client: PKCE is what proves a code is its own.
func New(issuer string) http.Handler {
	body, err := json.Marshal(metadata{
		Issuer:                            issuer,
		AuthorizationEndpoint:             issuer + oauthcontroller.AuthorizePath,
		TokenEndpoint:                     issuer + oauthcontroller.TokenPath,
		RegistrationEndpoint:              issuer + oauthcontroller.RegisterPath,
		ResponseTypes:                     []string{"code"},
		GrantTypes:                        []string{"authorization_code", "refresh_token"},
		CodeChallengeMethods:              []string{"S256"},
		TokenEndpointAuthMethods:          []string{"none"},
		AuthorizationResponseIssParameter: true,
	})
	if err != nil {
		panic(err) // a struct of strings always encodes.
	}

	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "public, max-age=3600")
		_, _ = w.Write(body)
	})
}
