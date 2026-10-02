package signin

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
)

const FlowCookieName = "cp_oauth"

const FlowTTL = 10 * time.Minute

type Flow struct {
	Provider  string
	State     string
	Verifier  string
	Nonce     string
	ExpiresAt time.Time
	Intent    accounts.Intent
	Account   accounts.AccountID
}

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

func (f *Flow) Challenge() string {
	sum := sha256.Sum256([]byte(f.Verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func (f *Flow) CallbackError(state string, now time.Time) error {
	if !now.Before(f.ExpiresAt) {
		return fmt.Errorf("%w: it lapsed at %s", ErrFlowInvalid, f.ExpiresAt.Format(time.RFC3339))
	}
	if subtle.ConstantTimeCompare([]byte(f.State), []byte(state)) != 1 {
		return fmt.Errorf("%w: the state does not match", ErrFlowInvalid)
	}
	return nil
}

func (f *Flow) AccountError(current *accounts.Account) error {
	if f.Intent != accounts.IntentLink {
		return nil
	}
	if current == nil || current.ID != f.Account {
		return fmt.Errorf("%w: the browser left the account the link started on", ErrFlowInvalid)
	}
	return nil
}

func (f *Flow) Cookie(sealed string, now time.Time) string {
	return accounts.Cookie(FlowCookieName, sealed, f.ExpiresAt, now)
}

func ExpiredFlowCookie() string {
	return accounts.ExpiredCookie(FlowCookieName)
}
