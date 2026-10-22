package exchange_code_usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/admintokens"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/connections"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Sealer interface {
	OpenedCode(sealed string) (*connections.Code, error)
}

// A code is good once: Spend refuses one it was handed before with ErrInvalidGrant.
type Spent interface {
	Spend(id string, until, now time.Time) error
}

type Tokens interface {
	SaveTokens(ctx context.Context, tokens []admintokens.Snapshot) error
}

func New(sealer Sealer, spent Spent, issuer *admintokens.Issuer, tokens Tokens, clock cptime.Clock) *UseCase {
	return &UseCase{sealer: sealer, spent: spent, issuer: issuer, tokens: tokens, clock: clock}
}

type UseCase struct {
	sealer Sealer
	spent  Spent
	issuer *admintokens.Issuer
	tokens Tokens
	clock  cptime.Clock
}

type In struct {
	Code       string
	Redemption connections.Redemption
}

func (u *UseCase) Execute(ctx context.Context, in In) (*admintokens.Pair, error) {
	now := u.clock.Now()

	code, err := u.sealer.OpenedCode(in.Code)
	if err != nil {
		return nil, fmt.Errorf("failed to read the code: %w", err)
	}
	if err := code.RedeemError(in.Redemption, now); err != nil {
		return nil, fmt.Errorf("failed to redeem the code: %w", err)
	}
	if err := u.spent.Spend(code.ID(), code.ExpiresAt(), now); err != nil {
		return nil, fmt.Errorf("failed to redeem the code: %w", err)
	}

	pair, err := u.issuer.Pair(code.Holder(), now)
	if err != nil {
		return nil, fmt.Errorf("failed to issue the tokens: %w", err)
	}
	if err := u.tokens.SaveTokens(ctx, pair.TakeSnapshots()); err != nil {
		return nil, fmt.Errorf("failed to keep the tokens: %w", err)
	}

	return pair, nil
}
