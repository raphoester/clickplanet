package google_identity_provider

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/signin"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/signin/oauth_http"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Endpoints struct {
	Authorize string
	Token     string
}

var Production = Endpoints{ //nolint:gosec // G101: public endpoint URLs, no credential in them
	Authorize: "https://accounts.google.com/o/oauth2/v2/auth",
	Token:     "https://oauth2.googleapis.com/token",
}

var issuers = []string{"https://accounts.google.com", "accounts.google.com"}

type Provider struct {
	client      signin.Client
	redirectURL string
	endpoints   Endpoints
	http        *http.Client
	clock       cptime.Clock
}

var _ signin.Provider = (*Provider)(nil)

func New(client signin.Client, redirectURL string, endpoints Endpoints, httpClient *http.Client, clock cptime.Clock) *Provider {
	return &Provider{client: client, redirectURL: redirectURL, endpoints: endpoints, http: httpClient, clock: clock}
}

func (p *Provider) AuthorizationURL(flow *signin.Flow) string {
	query := url.Values{
		"client_id":             {p.client.ClientID},
		"redirect_uri":          {p.redirectURL},
		"response_type":         {"code"},
		"scope":                 {"openid email"},
		"state":                 {flow.State()},
		"nonce":                 {flow.Nonce()},
		"code_challenge":        {flow.Challenge()},
		"code_challenge_method": {"S256"},
		"prompt":                {"select_account"},
	}
	return p.endpoints.Authorize + "?" + query.Encode()
}

type idToken struct {
	Issuer        string   `json:"iss"`
	Audience      audience `json:"aud"`
	ExpiresAt     int64    `json:"exp"`
	Nonce         string   `json:"nonce"`
	Subject       string   `json:"sub"`
	Email         string   `json:"email"`
	EmailVerified bool     `json:"email_verified"`
}

type audience []string

func (a *audience) UnmarshalJSON(raw []byte) error {
	var one string
	if err := json.Unmarshal(raw, &one); err == nil {
		*a = audience{one}
		return nil
	}
	var many []string
	if err := json.Unmarshal(raw, &many); err != nil {
		return fmt.Errorf("the audience is neither a string nor a list: %w", err)
	}
	*a = many
	return nil
}

func (p *Provider) Exchange(ctx context.Context, code string, flow *signin.Flow) (*accounts.Claim, error) {
	token, err := oauth_http.ExchangeCode(ctx, p.http, p.endpoints.Token, url.Values{
		"code":          {code},
		"client_id":     {p.client.ClientID},
		"client_secret": {p.client.ClientSecret},
		"redirect_uri":  {p.redirectURL},
		"code_verifier": {flow.Verifier()},
	})
	if err != nil {
		return nil, fmt.Errorf("google: %w", err)
	}

	claims, err := p.claimsOf(token.IDToken, flow)
	if err != nil {
		return nil, fmt.Errorf("%w: google: %w", signin.ErrProviderRefused, err)
	}
	claim := accounts.ClaimOf(claims.Subject, claims.Email, claims.EmailVerified)
	return &claim, nil
}

// Signature unchecked: safe only for a token fetched from Google's token endpoint over TLS.
func (p *Provider) claimsOf(raw string, flow *signin.Flow) (*idToken, error) {
	parts := strings.Split(raw, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("the ID token has %d parts, want 3", len(parts))
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("the ID token payload is not base64url: %w", err)
	}

	var claims idToken
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, fmt.Errorf("the ID token payload does not decode: %w", err)
	}

	switch {
	case !slices.Contains(issuers, claims.Issuer):
		return nil, fmt.Errorf("the ID token is from %q", claims.Issuer)
	case !slices.Contains(claims.Audience, p.client.ClientID):
		return nil, fmt.Errorf("the ID token is for %v", []string(claims.Audience))
	case !p.clock.Now().Before(time.Unix(claims.ExpiresAt, 0)):
		return nil, fmt.Errorf("the ID token expired at %d", claims.ExpiresAt)
	case claims.Nonce != flow.Nonce():
		return nil, fmt.Errorf("the ID token is for another sign-in")
	case claims.Subject == "":
		return nil, fmt.Errorf("the ID token names no user")
	}
	return &claims, nil
}
