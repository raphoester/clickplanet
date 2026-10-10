package access

import (
	"context"
	"errors"
	"fmt"

	"github.com/coreos/go-oidc/v3/oidc"
)

var ErrAnonymous = errors.New("the access assertion names no caller")

type Issuer string

type Audience string

type CloudflareVerifier struct {
	verifier *oidc.IDTokenVerifier
}

// ctx bounds the key set's background fetches, so it must outlive the call.
func NewCloudflareVerifier(ctx context.Context, issuer Issuer, audience Audience) *CloudflareVerifier {
	keys := oidc.NewRemoteKeySet(ctx, string(issuer)+"/cdn-cgi/access/certs")
	return &CloudflareVerifier{
		verifier: oidc.NewVerifier(string(issuer), keys, &oidc.Config{ClientID: string(audience)}),
	}
}

func (v *CloudflareVerifier) Verify(ctx context.Context, assertion string) (Caller, error) {
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
		return Caller(claims.CommonName), nil
	case claims.Email != "":
		return Caller(claims.Email), nil
	default:
		return "", ErrAnonymous
	}
}
