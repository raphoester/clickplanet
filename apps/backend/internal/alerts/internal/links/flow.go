package links

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
)

type AccountID = cpsession.AccountID

var ErrInvalidAccount = errors.New("not an account id")

func AccountIDOf(value string) (AccountID, error) {
	id, err := uuid.Parse(value)
	if err != nil || id == uuid.Nil {
		return AccountID{}, fmt.Errorf("%w: %q", ErrInvalidAccount, value)
	}
	return AccountID(id), nil
}

var (
	ErrNoFlow      = errors.New("this account started no Discord link")
	ErrFlowInvalid = errors.New("no Discord link this account started matches")
)

const FlowTTL = 10 * time.Minute

type Secrets interface {
	NewSecret() (string, error)
}

// One per account: a second start replaces the first.
type Flow struct {
	account   AccountID
	state     string
	verifier  string
	expiresAt time.Time
	spent     bool
}

func NewFlow(account AccountID, secrets Secrets, now time.Time) (Flow, error) {
	state, err := secrets.NewSecret()
	if err != nil {
		return Flow{}, fmt.Errorf("failed to draw the state: %w", err)
	}
	verifier, err := secrets.NewSecret()
	if err != nil {
		return Flow{}, fmt.Errorf("failed to draw the verifier: %w", err)
	}
	return Flow{account: account, state: state, verifier: verifier, expiresAt: now.Add(FlowTTL)}, nil
}

type Authorization struct {
	State     string
	Challenge string
}

func (f Flow) Authorization() Authorization {
	sum := sha256.Sum256([]byte(f.verifier))
	return Authorization{State: f.state, Challenge: base64.RawURLEncoding.EncodeToString(sum[:])}
}

// Good once: it answers the PKCE verifier the code is traded with.
func (f *Flow) Spend(state string, now time.Time) (string, error) {
	if f.spent || subtle.ConstantTimeCompare([]byte(state), []byte(f.state)) != 1 || !now.Before(f.expiresAt) {
		return "", ErrFlowInvalid
	}
	f.spent = true
	return f.verifier, nil
}

func (f *Flow) Forget() {
	f.spent = true
}

type FlowSnapshot struct {
	Account   AccountID
	State     string
	Verifier  string
	ExpiresAt time.Time
	Spent     bool
}

func (f Flow) TakeSnapshot() FlowSnapshot {
	return FlowSnapshot{Account: f.account, State: f.state, Verifier: f.verifier, ExpiresAt: f.expiresAt, Spent: f.spent}
}

func (s FlowSnapshot) Restore() Flow {
	return Flow{account: s.Account, state: s.State, verifier: s.Verifier, expiresAt: s.ExpiresAt, spent: s.Spent}
}
