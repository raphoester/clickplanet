package cloudflare_assertion_verifier

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"

	"github.com/raphoester/clickplanet.lol-backend/internal/ops/internal/access"
)

const keysTimeout = 10 * time.Second

type Config struct {
	Issuer   string
	Audience string
}

func (c Config) Validate() error {
	var audienceErr error
	if c.Audience == "" {
		audienceErr = errors.New("audience is empty: it is the AUD tag of the Cloudflare Access application")
	}

	issuer, err := url.Parse(c.Issuer)
	if err != nil || issuer.Scheme != "https" || issuer.Host == "" || issuer.String() != "https://"+issuer.Host {
		return errors.Join(audienceErr, fmt.Errorf(
			"issuer %q is not the team's address: want https://<team>.cloudflareaccess.com, with no path", c.Issuer,
		))
	}

	return audienceErr
}

func New(ctx context.Context, config Config) *Verifier {
	// The key set keeps this context for its values and drops its cancel, so the startup deadline does not end it.
	ctx = oidc.ClientContext(ctx, &http.Client{Timeout: keysTimeout})
	keys := oidc.NewRemoteKeySet(ctx, config.Issuer+"/cdn-cgi/access/certs")

	return &Verifier{verifier: oidc.NewVerifier(config.Issuer, keys, &oidc.Config{ClientID: config.Audience})}
}

type Verifier struct {
	verifier *oidc.IDTokenVerifier
}

func (v *Verifier) Caller(ctx context.Context, assertion string) (access.Caller, error) {
	token, err := v.verifier.Verify(ctx, assertion)
	if err != nil {
		return "", fmt.Errorf("failed to verify the access assertion: %w", err)
	}

	var claims struct {
		CommonName string `json:"common_name"`
		Email      string `json:"email"`
	}
	if err := token.Claims(&claims); err != nil {
		return "", fmt.Errorf("failed to read the access assertion: %w", err)
	}

	switch {
	case claims.CommonName != "":
		return access.Caller(claims.CommonName), nil
	case claims.Email != "":
		return access.Caller(claims.Email), nil
	default:
		return "", access.ErrAnonymous
	}
}
