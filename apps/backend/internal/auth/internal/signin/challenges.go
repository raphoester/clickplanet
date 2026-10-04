package signin

import (
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpratelimit"
)

type Limiter interface {
	Take(key string) (bool, cpratelimit.State)
}

type Challenges struct {
	offered bool
	secrets Secrets
	codes   Codes
	sealer  ChallengeSealer
	guesses Limiter
}

func NewChallenges(offered bool, secrets Secrets, codes Codes, sealer ChallengeSealer, guesses Limiter) *Challenges {
	return &Challenges{offered: offered, secrets: secrets, codes: codes, sealer: sealer, guesses: guesses}
}

func (c *Challenges) Off() bool {
	return !c.offered
}

func (c *Challenges) Issued(address Address, intent accounts.Intent, account accounts.AccountID, now time.Time) (*Challenge, string, error) {
	challenge, err := NewChallenge(address, intent, account, c.secrets, c.codes, now)
	if err != nil {
		return nil, "", err
	}
	sealed, err := c.sealer.SealedChallenge(challenge)
	if err != nil {
		return nil, "", fmt.Errorf("failed to seal the challenge: %w", err)
	}
	return challenge, challenge.Cookie(sealed, now), nil
}

func (c *Challenges) Opened(cookieHeader string) (*Challenge, error) {
	sealed, found := accounts.CookieValue(cookieHeader, ChallengeCookieName)
	if !found {
		return nil, fmt.Errorf("%w: the browser sent no %s cookie", ErrFlowInvalid, ChallengeCookieName)
	}
	challenge, err := c.sealer.OpenedChallenge(sealed)
	if err != nil {
		return nil, fmt.Errorf("failed to open the challenge: %w", err)
	}
	return challenge, nil
}

func (c *Challenges) Guess(challenge *Challenge, code string, now time.Time) error {
	if allowed, _ := c.guesses.Take(challenge.id); !allowed {
		return fmt.Errorf("%w: too many codes were wrong", ErrFlowInvalid)
	}
	return challenge.CodeError(code, now)
}
