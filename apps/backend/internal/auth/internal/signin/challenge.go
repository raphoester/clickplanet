package signin

import (
	"crypto/subtle"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
)

const (
	ChallengeCookieName = "cp_email"
	ChallengeTTL        = 10 * time.Minute
	// MaxAttempts is checked on the server: the browser sends the same sealed cookie with every guess.
	MaxAttempts = 5
)

// Challenge is one email sign-in in progress, sealed in the browser's cookie like a Flow.
type Challenge struct {
	// Keys the attempts budget.
	ID        string
	Address   Address
	Code      string
	ExpiresAt time.Time
	Intent    accounts.Intent
	Account   accounts.AccountID
}

type Codes interface {
	NewCode() (string, error)
}

// ChallengeSealer answers ErrFlowInvalid for anything it did not seal.
type ChallengeSealer interface {
	SealedChallenge(challenge *Challenge) (string, error)
	OpenedChallenge(sealed string) (*Challenge, error)
}

func NewChallenge(
	address Address, intent accounts.Intent, account accounts.AccountID, secrets Secrets, codes Codes, now time.Time,
) (*Challenge, error) {
	id, err := secrets.NewSecret()
	if err != nil {
		return nil, fmt.Errorf("failed to draw the challenge id: %w", err)
	}
	code, err := codes.NewCode()
	if err != nil {
		return nil, fmt.Errorf("failed to draw the code: %w", err)
	}
	return &Challenge{ID: id, Address: address, Code: code, ExpiresAt: now.Add(ChallengeTTL), Intent: intent, Account: account}, nil
}

// CodeError is ErrFlowInvalid for a lapsed challenge and ErrWrongCode for another code.
func (c *Challenge) CodeError(code string, now time.Time) error {
	if !now.Before(c.ExpiresAt) {
		return fmt.Errorf("%w: it lapsed at %s", ErrFlowInvalid, c.ExpiresAt.Format(time.RFC3339))
	}
	if subtle.ConstantTimeCompare([]byte(c.Code), []byte(code)) != 1 {
		return ErrWrongCode
	}
	return nil
}

func (c *Challenge) AccountError(current *accounts.Account) error {
	return accountError(c.Intent, c.Account, current)
}

// Claim is the address, verified: the player typed the code that was sent to it.
func (c *Challenge) Claim() accounts.Claim {
	return accounts.Claim{Subject: string(c.Address), Email: string(c.Address), EmailVerified: true}
}

func (c *Challenge) Cookie(sealed string, now time.Time) string {
	return accounts.Cookie(ChallengeCookieName, sealed, c.ExpiresAt, now)
}

func ExpiredChallengeCookie() string {
	return accounts.ExpiredCookie(ChallengeCookieName)
}
