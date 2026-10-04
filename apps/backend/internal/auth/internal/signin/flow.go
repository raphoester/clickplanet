package signin

import (
	"context"
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
	provider  string
	state     string
	verifier  string
	nonce     string
	expiresAt time.Time
	intent    accounts.Intent
	account   accounts.AccountID
}

type Secrets interface {
	NewSecret() (string, error)
}

func NewFlow(provider string, intent accounts.Intent, account accounts.AccountID, secrets Secrets, now time.Time) (*Flow, error) {
	flow := &Flow{provider: provider, expiresAt: now.Add(FlowTTL), intent: intent, account: account}
	for _, field := range []*string{&flow.state, &flow.verifier, &flow.nonce} {
		secret, err := secrets.NewSecret()
		if err != nil {
			return nil, fmt.Errorf("failed to draw a secret: %w", err)
		}
		*field = secret
	}
	return flow, nil
}

func FlowOf(
	provider string, state string, verifier string, nonce string, expiresAt time.Time, intent accounts.Intent, account accounts.AccountID,
) *Flow {
	return &Flow{
		provider: provider, state: state, verifier: verifier, nonce: nonce, expiresAt: expiresAt, intent: intent, account: account,
	}
}

func (f *Flow) Provider() string {
	return f.provider
}

func (f *Flow) State() string {
	return f.state
}

func (f *Flow) Verifier() string {
	return f.verifier
}

func (f *Flow) Nonce() string {
	return f.nonce
}

func (f *Flow) ExpiresAt() time.Time {
	return f.expiresAt
}

func (f *Flow) Intent() accounts.Intent {
	return f.intent
}

func (f *Flow) Account() accounts.AccountID {
	return f.account
}

func (f *Flow) Challenge() string {
	sum := sha256.Sum256([]byte(f.verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func (f *Flow) CallbackError(state string, now time.Time) error {
	if !now.Before(f.expiresAt) {
		return fmt.Errorf("%w: it lapsed at %s", ErrFlowInvalid, f.expiresAt.Format(time.RFC3339))
	}
	if subtle.ConstantTimeCompare([]byte(f.state), []byte(state)) != 1 {
		return fmt.Errorf("%w: the state does not match", ErrFlowInvalid)
	}
	return nil
}

func (f *Flow) AccountError(current *accounts.Account) error {
	return accountError(f.intent, f.account, current)
}

func LinkTarget(
	ctx context.Context, sessions accounts.SessionFinder, intent accounts.Intent, cookieHeader string, now time.Time,
) (accounts.AccountID, error) {
	if intent != accounts.IntentLink {
		return accounts.AccountID{}, nil
	}
	session, err := accounts.Caller(ctx, sessions, cookieHeader, now)
	if err != nil {
		return accounts.AccountID{}, fmt.Errorf("failed to find the account to link to: %w", err)
	}
	return session.Account(), nil
}

func accountError(intent accounts.Intent, started accounts.AccountID, current *accounts.Account) error {
	if intent != accounts.IntentLink {
		return nil
	}
	if current == nil || current.ID() != started {
		return fmt.Errorf("%w: the browser left the account the link started on", ErrFlowInvalid)
	}
	return nil
}

func (f *Flow) Cookie(sealed string, now time.Time) string {
	return accounts.Cookie(FlowCookieName, sealed, f.expiresAt, now)
}

func ExpiredFlowCookie() string {
	return accounts.ExpiredCookie(FlowCookieName)
}
