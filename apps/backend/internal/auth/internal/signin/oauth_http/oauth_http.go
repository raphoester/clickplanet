// Package oauth_http is the two calls every OAuth 2.0 provider shares: trade a code for a token, and read a JSON resource with it.
package oauth_http

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/signin"
)

const maxBodyBytes = 1 << 16

// Token is the part of a token endpoint's answer this server reads.
type Token struct {
	AccessToken string `json:"access_token"`
	IDToken     string `json:"id_token"`
}

// ExchangeCode posts the authorization code grant, PKCE verifier included.
func ExchangeCode(ctx context.Context, client *http.Client, endpoint string, form url.Values) (*Token, error) {
	form.Set("grant_type", "authorization_code")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to build the token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	token, err := answer[Token](client, req)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange the code: %w", err)
	}
	if token.AccessToken == "" {
		return nil, fmt.Errorf("%w: the token endpoint answered no access token", signin.ErrProviderRefused)
	}
	return token, nil
}

// Resource is the JSON resource at endpoint, read with the access token.
func Resource[T any](ctx context.Context, client *http.Client, endpoint string, accessToken string) (*T, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to build the request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")

	return answer[T](client, req)
}

// answer is the provider's decoded answer to req: ErrProviderRefused for a 4xx or a body that does not decode, a plain error when the provider could not be asked.
func answer[T any](client *http.Client, req *http.Request) (*T, error) {
	res, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("the provider is unreachable: %w", err)
	}
	defer func() { _ = res.Body.Close() }()

	body := io.LimitReader(res.Body, maxBodyBytes)
	if res.StatusCode >= http.StatusInternalServerError {
		return nil, fmt.Errorf("the provider answered %d", res.StatusCode)
	}
	if res.StatusCode != http.StatusOK {
		var reason struct {
			Error string `json:"error"`
		}
		_ = json.NewDecoder(body).Decode(&reason)
		return nil, fmt.Errorf("%w: it answered %d (%s)", signin.ErrProviderRefused, res.StatusCode, reason.Error)
	}

	var decoded T
	if err := json.NewDecoder(body).Decode(&decoded); err != nil {
		return nil, fmt.Errorf("%w: its answer is not JSON: %w", signin.ErrProviderRefused, err)
	}
	return &decoded, nil
}
