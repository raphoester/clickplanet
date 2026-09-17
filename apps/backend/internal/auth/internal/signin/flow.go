// Package signin is how a browser proves it holds a provider's user: OAuth 2.0 with PKCE, run by this server.
package signin

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
)

// FlowCookieName holds a sign-in between StartSignIn and CompleteSignIn.
const FlowCookieName = "cp_oauth"

// FlowTTL is how long a player has to come back from the provider.
const FlowTTL = 10 * time.Minute

// Flow is one sign-in in progress. It lives sealed in the browser's cookie, never on the server.
type Flow struct {
	Provider string
	// Sent to the provider and back; a callback whose state is not this one was not started by this browser.
	State string
	// The PKCE secret: only its challenge goes to the provider, so an intercepted code is worth nothing.
	Verifier string
	// Signed into Google's ID token, so a token from another sign-in is refused.
	Nonce     string
	ExpiresAt time.Time
	Intent    accounts.Intent
	// The account a link started on, which must still be the browser's at the callback. Zero for a sign-in.
	Account accounts.AccountID
}

// Secrets draws the random strings a flow is made of.
type Secrets interface {
	NewSecret() (string, error)
}

func NewFlow(provider string, intent accounts.Intent, account accounts.AccountID, secrets Secrets, now time.Time) (*Flow, error) {
	flow := &Flow{Provider: provider, ExpiresAt: now.Add(FlowTTL), Intent: intent, Account: account}
	for _, field := range []*string{&flow.State, &flow.Verifier, &flow.Nonce} {
		secret, err := secrets.NewSecret()
		if err != nil {
			return nil, fmt.Errorf("failed to draw a secret: %w", err)
		}
		*field = secret
	}
	return flow, nil
}

// Challenge is the PKCE S256 challenge of the verifier.
func (f *Flow) Challenge() string {
	sum := sha256.Sum256([]byte(f.Verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

// CallbackError refuses a flow that has lapsed, or a callback that carries another state.
func (f *Flow) CallbackError(state string, now time.Time) error {
	if !now.Before(f.ExpiresAt) {
		return fmt.Errorf("%w: it lapsed at %s", ErrFlowInvalid, f.ExpiresAt.Format(time.RFC3339))
	}
	if subtle.ConstantTimeCompare([]byte(f.State), []byte(state)) != 1 {
		return fmt.Errorf("%w: the state does not match", ErrFlowInvalid)
	}
	return nil
}

// AccountError refuses a link whose browser is no longer on the account it started on: signed out, or signed in elsewhere since.
func (f *Flow) AccountError(current *accounts.Account) error {
	if f.Intent != accounts.IntentLink {
		return nil
	}
	if current == nil || current.ID != f.Account {
		return fmt.Errorf("%w: the browser left the account the link started on", ErrFlowInvalid)
	}
	return nil
}

// Cookie keeps the sealed flow in the browser until it lapses.
func (f *Flow) Cookie(sealed string, now time.Time) string {
	return accounts.Cookie(FlowCookieName, sealed, f.ExpiresAt, now)
}

// ClearFlowCookie is the Set-Cookie that ends a sign-in, whatever became of it.
func ExpiredFlowCookie() string {
	return accounts.ExpiredCookie(FlowCookieName)
}
