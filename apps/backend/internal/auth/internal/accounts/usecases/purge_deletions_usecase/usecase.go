package purge_deletions_usecase

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

type Purger interface {
	PurgeDeletions(ctx context.Context, requestedBefore time.Time, limit int) ([]accounts.AccountID, error)
}

type Executor interface {
	Execute(ctx context.Context) (int, error)
}

type Config struct {
	// How long an account whose deletion was requested is kept, for a sign-in to bring it back. The privacy policy names
	// it: a change is a change to the policy too.
	Grace time.Duration

	Interval time.Duration
}

const (
	defaultGrace    = 30 * 24 * time.Hour
	defaultInterval = time.Hour
	batch           = 1000
	batchTimeout    = 30 * time.Second
)

func (c Config) WithDefaults() Config {
	if c.Grace <= 0 {
		c.Grace = defaultGrace
	}
	if c.Interval <= 0 {
		c.Interval = defaultInterval
	}
	return c
}

type UseCase struct {
	config        Config
	transactioner cpatomic.Transactioner
	accounts      Purger
	outbox        cpatomic.Outbox
	clock         cptime.Clock
}

var _ Executor = (*UseCase)(nil)

func New(
	config Config, transactioner cpatomic.Transactioner, accounts Purger, outbox cpatomic.Outbox, clock cptime.Clock,
) *UseCase {
	return &UseCase{config: config.WithDefaults(), transactioner: transactioner, accounts: accounts, outbox: outbox, clock: clock}
}

// Deletes the accounts whose deletion was requested longer ago than the grace period, and tells the other modules,
// which delete what they keep about them.
func (u *UseCase) Execute(ctx context.Context) (int, error) {
	requestedBefore := u.clock.Now().Add(-u.config.Grace)

	total := 0
	for {
		purged, err := u.purgeBatch(ctx, requestedBefore)
		total += purged
		if err != nil {
			return total, err
		}
		if purged < batch {
			return total, nil
		}
	}
}

func (u *UseCase) purgeBatch(ctx context.Context, requestedBefore time.Time) (int, error) {
	ctx, cancel := context.WithTimeout(ctx, batchTimeout)
	defer cancel()

	var purged []accounts.AccountID
	err := u.transactioner.InTx(ctx, func(ctx context.Context) error {
		var err error
		if purged, err = u.accounts.PurgeDeletions(ctx, requestedBefore, batch); err != nil {
			return fmt.Errorf("failed to purge a batch: %w", err)
		}
		deleted := make([]proto.Message, len(purged))
		for i, account := range purged {
			deleted[i] = &authv1.AccountDeleted{AccountId: account.String()}
		}
		if err := u.outbox.Record(ctx, deleted...); err != nil {
			return fmt.Errorf("failed to record the purged accounts: %w", err)
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	return len(purged), nil
}
