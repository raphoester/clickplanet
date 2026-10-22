package refresh_tokens_usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/admintokens"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/connections"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Tokens interface {
	SaveTokens(ctx context.Context, tokens []admintokens.Snapshot) error
	SpendRefresh(ctx context.Context, hash accounts.TokenHash, now time.Time) (admintokens.Holder, error)
}

func New(issuer *admintokens.Issuer, tokens Tokens, clock cptime.Clock) *UseCase {
	return &UseCase{issuer: issuer, tokens: tokens, clock: clock}
}

type UseCase struct {
	issuer *admintokens.Issuer
	tokens Tokens
	clock  cptime.Clock
}

type In struct {
	RefreshToken admintokens.Secret
	ClientID     string
}

// OAuth 2.1 rotation: the refresh token is spent, and the pair that replaces it carries a new one. A refresh token
// sent by another client is spent too, and buys nothing: whoever sent it should not hold it.
func (u *UseCase) Execute(ctx context.Context, in In) (*admintokens.Pair, error) {
	now := u.clock.Now()

	holder, err := u.tokens.SpendRefresh(ctx, in.RefreshToken.Hash(), now)
	if err != nil {
		return nil, fmt.Errorf("failed to spend the refresh token: %w", err)
	}
	if holder.ClientID != connections.IDOf(in.ClientID) {
		return nil, fmt.Errorf("%w: issued to another client", admintokens.ErrUnknownToken)
	}

	pair, err := u.issuer.Pair(holder, now)
	if err != nil {
		return nil, fmt.Errorf("failed to issue the tokens: %w", err)
	}
	if err := u.tokens.SaveTokens(ctx, pair.TakeSnapshots()); err != nil {
		return nil, fmt.Errorf("failed to keep the tokens: %w", err)
	}

	return pair, nil
}
