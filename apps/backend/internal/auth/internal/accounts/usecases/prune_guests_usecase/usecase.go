package prune_guests_usecase

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/protobuf/proto"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Pruner interface {
	PruneGuests(ctx context.Context, idleSince time.Time, limit int) ([]accounts.AccountID, error)
}

type Executor interface {
	Execute(ctx context.Context) (int, error)
}

type Config struct {
	IdleFor time.Duration

	Interval time.Duration
}

const (
	defaultIdleFor  = 90 * 24 * time.Hour
	defaultInterval = time.Hour
	batch           = 1000
	batchTimeout    = 30 * time.Second
)

func (c Config) WithDefaults() Config {
	if c.IdleFor <= 0 {
		c.IdleFor = defaultIdleFor
	}
	if c.Interval <= 0 {
		c.Interval = defaultInterval
	}
	return c
}

type UseCase struct {
	config        Config
	transactioner cpatomic.Transactioner
	guests        Pruner
	outbox        cpatomic.Outbox
	clock         cptime.Clock
}

var _ Executor = (*UseCase)(nil)

func New(
	config Config, transactioner cpatomic.Transactioner, guests Pruner, outbox cpatomic.Outbox, clock cptime.Clock,
) *UseCase {
	return &UseCase{config: config.WithDefaults(), transactioner: transactioner, guests: guests, outbox: outbox, clock: clock}
}

func (u *UseCase) Execute(ctx context.Context) (int, error) {
	idleSince := u.clock.Now().Add(-u.config.IdleFor)

	total := 0
	for {
		pruned, err := u.pruneBatch(ctx, idleSince)
		total += pruned
		if err != nil {
			return total, err
		}
		if pruned < batch {
			return total, nil
		}
	}
}

func (u *UseCase) pruneBatch(ctx context.Context, idleSince time.Time) (int, error) {
	ctx, cancel := context.WithTimeout(ctx, batchTimeout)
	defer cancel()

	var pruned []accounts.AccountID
	err := u.transactioner.InTx(ctx, func(ctx context.Context) error {
		var err error
		if pruned, err = u.guests.PruneGuests(ctx, idleSince, batch); err != nil {
			return fmt.Errorf("failed to prune a batch: %w", err)
		}
		deleted := make([]proto.Message, len(pruned))
		for i, account := range pruned {
			deleted[i] = &authv1.AccountDeleted{AccountId: account.String()}
		}
		if err := u.outbox.Record(ctx, deleted...); err != nil {
			return fmt.Errorf("failed to record the pruned guests: %w", err)
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	return len(pruned), nil
}
