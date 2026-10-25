package prune_guests_usecase

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/protobuf/proto"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Pruner interface {
	PruneGuests(ctx context.Context, idleSince time.Time, limit int, told func(accounts.AccountID) proto.Message) ([]accounts.AccountID, error)
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
	config Config
	guests Pruner
	clock  cptime.Clock
}

var _ Executor = (*UseCase)(nil)

func New(config Config, guests Pruner, clock cptime.Clock) *UseCase {
	return &UseCase{config: config.WithDefaults(), guests: guests, clock: clock}
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

	pruned, err := u.guests.PruneGuests(ctx, idleSince, batch, deletedOf)
	if err != nil {
		return 0, fmt.Errorf("failed to prune a batch: %w", err)
	}
	return len(pruned), nil
}

func deletedOf(account accounts.AccountID) proto.Message {
	return &authv1.AccountDeleted{AccountId: account.String()}
}
