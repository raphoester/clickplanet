package anonymize_takes_usecase

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger"
)

type Ledger interface {
	Flush(ctx context.Context) error
}

type Takes interface {
	AnonymizeTakes(ctx context.Context, account ledger.AccountID) error
}

type Tiles interface {
	ForgetHolder(ctx context.Context, account ledger.AccountID) (int, error)
}

// Arenas keep the map from before each arena open, holders too.
type Arenas interface {
	ForgetHolder(ctx context.Context, account ledger.AccountID) error
}

type UseCase struct {
	ledger Ledger
	takes  Takes
	arenas Arenas
	tiles  Tiles
}

func New(ledger Ledger, takes Takes, arenas Arenas, tiles Tiles) *UseCase {
	return &UseCase{ledger: ledger, takes: takes, arenas: arenas, tiles: tiles}
}

func (u *UseCase) Execute(ctx context.Context, account ledger.AccountID) error {
	// A take still in memory would be flushed after the update and name the account again.
	if err := u.ledger.Flush(ctx); err != nil {
		return fmt.Errorf("failed to flush the ledger: %w", err)
	}
	if err := u.takes.AnonymizeTakes(ctx, account); err != nil {
		return fmt.Errorf("failed to anonymize the takes: %w", err)
	}
	// Before the map: an arena closed in between brings back a map that names it no more.
	if err := u.arenas.ForgetHolder(ctx, account); err != nil {
		return fmt.Errorf("failed to forget the holder in the arenas: %w", err)
	}
	if _, err := u.tiles.ForgetHolder(ctx, account); err != nil {
		return fmt.Errorf("failed to forget the tiles' holder: %w", err)
	}
	return nil
}
