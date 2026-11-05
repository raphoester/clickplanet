package ban_player_usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/bans"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/guard"
)

var ErrNegativeDuration = errors.New("a ban cannot last a negative time")

type Banner interface {
	Ban(ctx context.Context, scope, account string, duration time.Duration) (guard.Sentence, error)
	Enforcing() bool
}

type In struct {
	Scope    string
	Account  string
	Duration time.Duration
}

type Out struct {
	Scope    string
	Account  string
	Offence  int
	Until    time.Time
	Enforced bool
}

func New(banner Banner) *UseCase {
	return &UseCase{banner: banner}
}

type UseCase struct {
	banner Banner
}

func (u *UseCase) Execute(ctx context.Context, in In) (Out, error) {
	caller, err := bans.ParseCaller(in.Scope, in.Account)
	if err != nil {
		return Out{}, fmt.Errorf("cannot ban: %w", err)
	}
	if in.Duration < 0 {
		return Out{}, fmt.Errorf("%w: %s", ErrNegativeDuration, in.Duration)
	}

	sentence, err := u.banner.Ban(ctx, caller.Scope, caller.Account, in.Duration)
	if err != nil {
		return Out{}, fmt.Errorf("failed to ban: %w", err)
	}

	return Out{
		Scope:    caller.Scope,
		Account:  caller.Account,
		Offence:  sentence.Offence,
		Until:    sentence.Until,
		Enforced: u.banner.Enforcing(),
	}, nil
}
