// Package ban_player is the operator's shadow ban: the same sentence the antibot passes, on a scope a person picked.
package ban_player

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpipscope"
)

var (
	ErrNegativeDuration = errors.New("a ban cannot last a negative time")
	ErrAntiBotOff       = errors.New("antiBot is off, so there is no ban to pass")
)

type Banner interface {
	Ban(scope string, duration time.Duration) antibot.Sentence
	Enforcing() bool
}

type In struct {
	// Scope is a scope as FindPlayers lists it, or any address, which is banned as its scope.
	Scope string
	// Zero takes the ladder's step for the offence.
	Duration time.Duration
}

type Out struct {
	Scope    string
	Offence  int
	Until    time.Time
	Enforced bool
}

// New takes a nil banner when the antibot is off.
func New(banner Banner) *UseCase {
	return &UseCase{banner: banner}
}

type UseCase struct {
	banner Banner
}

func (u *UseCase) Execute(_ context.Context, in In) (Out, error) {
	if u.banner == nil {
		return Out{}, ErrAntiBotOff
	}

	scope, ok := cpipscope.Parse(in.Scope)
	if !ok {
		return Out{}, fmt.Errorf("%w: %q", clicks.ErrInvalidScope, in.Scope)
	}
	if in.Duration < 0 {
		return Out{}, fmt.Errorf("%w: %s", ErrNegativeDuration, in.Duration)
	}

	sentence := u.banner.Ban(scope, in.Duration)

	return Out{Scope: scope, Offence: sentence.Offence, Until: sentence.Until, Enforced: u.banner.Enforcing()}, nil
}
