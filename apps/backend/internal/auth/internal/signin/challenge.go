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
	// Counted on the server: the browser sends the same sealed cookie with every guess.
	MaxAttempts = 5
)

type Challenge struct {
	id        string
	address   Address
	code      string
	expiresAt time.Time
	intent    accounts.Intent
	account   accounts.AccountID
}

type Codes interface {
	NewCode() (string, error)
}

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
	return &Challenge{id: id, address: address, code: code, expiresAt: now.Add(ChallengeTTL), intent: intent, account: account}, nil
}

func ChallengeOf(
	id string, address Address, code string, expiresAt time.Time, intent accounts.Intent, account accounts.AccountID,
) *Challenge {
	return &Challenge{id: id, address: address, code: code, expiresAt: expiresAt, intent: intent, account: account}
}

func (c *Challenge) ID() string {
	return c.id
}

func (c *Challenge) Address() Address {
	return c.address
}

func (c *Challenge) Code() string {
	return c.code
}

func (c *Challenge) ExpiresAt() time.Time {
	return c.expiresAt
}

func (c *Challenge) Intent() accounts.Intent {
	return c.intent
}

func (c *Challenge) Account() accounts.AccountID {
	return c.account
}

func (c *Challenge) CodeError(code string, now time.Time) error {
	if !now.Before(c.expiresAt) {
		return fmt.Errorf("%w: it lapsed at %s", ErrFlowInvalid, c.expiresAt.Format(time.RFC3339))
	}
	if subtle.ConstantTimeCompare([]byte(c.code), []byte(code)) != 1 {
		return ErrWrongCode
	}
	return nil
}

func (c *Challenge) AccountError(current *accounts.Account) error {
	return accountError(c.intent, c.account, current)
}

func (c *Challenge) Letter() Letter {
	return CodeLetter(c.code)
}

func (c *Challenge) Claim() accounts.Claim {
	return accounts.ClaimOf(string(c.address), string(c.address), true)
}

func (c *Challenge) Cookie(sealed string, now time.Time) string {
	return accounts.Cookie(ChallengeCookieName, sealed, c.expiresAt, now)
}

func ExpiredChallengeCookie() string {
	return accounts.ExpiredCookie(ChallengeCookieName)
}
