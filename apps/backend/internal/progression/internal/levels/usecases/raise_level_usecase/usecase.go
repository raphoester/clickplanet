package raise_level_usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	progressionv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/progression/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/levels"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/stats"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Stats interface {
	Stats(ctx context.Context, account accounts.AccountID) (stats.Stats, error)
}

type Accounts interface {
	Account(ctx context.Context, account accounts.AccountID) (accounts.Account, error)
}

type Levels interface {
	Levels(ctx context.Context, accounts []accounts.AccountID) (levels.ByAccount, error)
	Raise(ctx context.Context, account accounts.AccountID, level levels.Level, at time.Time) (levels.Level, error)
}

type UseCase struct {
	stats         Stats
	accounts      Accounts
	levels        Levels
	transactioner cpatomic.Transactioner
	outbox        cpatomic.Outbox
	clock         cptime.Clock
	visibility    levels.Visibility
}

func New(
	stats Stats, accounts Accounts, levels Levels, transactioner cpatomic.Transactioner, outbox cpatomic.Outbox,
	clock cptime.Clock, visibility levels.Visibility,
) *UseCase {
	return &UseCase{
		stats: stats, accounts: accounts, levels: levels, transactioner: transactioner, outbox: outbox, clock: clock,
		visibility: visibility,
	}
}

// Execute answers the level it raised the account to, 0 when it raised none.
func (u *UseCase) Execute(ctx context.Context, account accounts.AccountID) (levels.Level, error) {
	counted, err := u.stats.Stats(ctx, account)
	if errors.Is(err, stats.ErrNoStats) {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("failed to read the stats: %w", err)
	}

	kept, err := u.levels.Levels(ctx, []accounts.AccountID{account})
	if err != nil {
		return 0, fmt.Errorf("failed to read the level kept: %w", err)
	}
	scale := levels.Scale{}
	reach := scale.Reach(counted)
	if reach <= kept[account] {
		return 0, nil
	}

	known, err := u.accounts.Account(ctx, account)
	if err != nil {
		return 0, fmt.Errorf("failed to ask player about the account: %w", err)
	}
	if scale.StandingOf(known, counted).Empty() {
		return 0, nil
	}

	at := u.clock.Now()
	var raised levels.Level
	err = u.transactioner.InTx(ctx, func(ctx context.Context) error {
		previous, err := u.levels.Raise(ctx, account, reach, at)
		if err != nil {
			return fmt.Errorf("failed to raise the level: %w", err)
		}
		if previous >= reach {
			return nil
		}
		raised = reach
		if err := u.outbox.Record(ctx, &progressionv1.LevelReached{
			AccountId: account.String(),
			Level:     uint32(reach),
			Previous:  uint32(previous),
			ReachedAt: timestamppb.New(at),
			Silent:    u.visibility.Silent(),
		}); err != nil {
			return fmt.Errorf("failed to record the level reached: %w", err)
		}
		return nil
	})
	if err != nil {
		return 0, err //nolint:wrapcheck // each step named it.
	}
	return raised, nil
}
