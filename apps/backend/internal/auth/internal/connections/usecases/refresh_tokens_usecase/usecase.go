package refresh_tokens_usecase

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/admintokens"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/connections"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Tokens interface {
	Connection(ctx context.Context, hash accounts.TokenHash) ([]*admintokens.Token, error)
	SaveTokens(ctx context.Context, tokens ...admintokens.Snapshot) error
}

func New(issuer *admintokens.Issuer, tokens Tokens, transactioner cpatomic.Transactioner, clock cptime.Clock) *UseCase {
	return &UseCase{issuer: issuer, tokens: tokens, transactioner: transactioner, clock: clock}
}

type UseCase struct {
	issuer        *admintokens.Issuer
	tokens        Tokens
	transactioner cpatomic.Transactioner
	clock         cptime.Clock
}

type In struct {
	RefreshToken admintokens.Secret
	ClientID     string
}

func (u *UseCase) Execute(ctx context.Context, in In) (*admintokens.Pair, error) {
	now := u.clock.Now()

	var trade admintokens.Trade
	err := u.transactioner.InTx(ctx, func(ctx context.Context) error {
		connection, err := u.tokens.Connection(ctx, in.RefreshToken.Hash())
		if err != nil {
			return fmt.Errorf("failed to read the connection: %w", err)
		}
		trade, err = u.issuer.Trade(connection, in.RefreshToken, connections.IDOf(in.ClientID), now)
		if err != nil {
			return fmt.Errorf("failed to issue the tokens: %w", err)
		}
		if err := u.tokens.SaveTokens(ctx, trade.Kept...); err != nil {
			return fmt.Errorf("failed to keep the tokens: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, err //nolint:wrapcheck // each step already says what failed.
	}
	if trade.Refusal != nil {
		return nil, fmt.Errorf("failed to trade the refresh token: %w", trade.Refusal)
	}

	return trade.Pair, nil
}
