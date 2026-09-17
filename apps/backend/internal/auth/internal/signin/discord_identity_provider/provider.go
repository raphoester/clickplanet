// Package discord_identity_provider signs a player in with Discord, over OAuth 2.0, and reads the user from /users/@me.
package discord_identity_provider

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/signin"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/signin/oauth_http"
)

type Endpoints struct {
	Authorize string
	Token     string
	Me        string
}

var Production = Endpoints{ //nolint:gosec // G101: public endpoint URLs, no credential in them
	Authorize: "https://discord.com/oauth2/authorize",
	Token:     "https://discord.com/api/oauth2/token",
	Me:        "https://discord.com/api/users/@me",
}

type Provider struct {
	client      signin.Client
	redirectURL string
	endpoints   Endpoints
	http        *http.Client
}

var _ signin.Provider = (*Provider)(nil)

func New(client signin.Client, redirectURL string, endpoints Endpoints, httpClient *http.Client) *Provider {
	return &Provider{client: client, redirectURL: redirectURL, endpoints: endpoints, http: httpClient}
}

func (p *Provider) AuthorizationURL(flow *signin.Flow) string {
	query := url.Values{
		"client_id":             {p.client.ClientID},
		"redirect_uri":          {p.redirectURL},
		"response_type":         {"code"},
		"scope":                 {"identify email"},
		"state":                 {flow.State},
		"code_challenge":        {flow.Challenge()},
		"code_challenge_method": {"S256"},
	}
	return p.endpoints.Authorize + "?" + query.Encode()
}

type user struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	// Whether the email is verified.
	Verified bool `json:"verified"`
}

func (p *Provider) Exchange(ctx context.Context, code string, flow *signin.Flow) (*accounts.Claim, error) {
	token, err := oauth_http.ExchangeCode(ctx, p.http, p.endpoints.Token, url.Values{
		"code":          {code},
		"client_id":     {p.client.ClientID},
		"client_secret": {p.client.ClientSecret},
		"redirect_uri":  {p.redirectURL},
		"code_verifier": {flow.Verifier},
	})
	if err != nil {
		return nil, fmt.Errorf("discord: %w", err)
	}

	me, err := oauth_http.Resource[user](ctx, p.http, p.endpoints.Me, token.AccessToken)
	if err != nil {
		return nil, fmt.Errorf("discord: failed to read the user: %w", err)
	}
	if me.ID == "" {
		return nil, fmt.Errorf("%w: discord named no user", signin.ErrProviderRefused)
	}
	return &accounts.Claim{Subject: me.ID, Email: me.Email, EmailVerified: me.Verified}, nil
}
