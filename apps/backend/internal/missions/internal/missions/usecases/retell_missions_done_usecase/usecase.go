package retell_missions_done_usecase

import (
	"context"
	"fmt"

	"google.golang.org/protobuf/proto"

	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions/donemessage"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
)

type Store interface {
	DoneAfter(ctx context.Context, after missions.AccountID, limit int) ([]missions.AccountDone, error)
}

type Executor interface {
	Execute(ctx context.Context, in In) (Out, error)
}

type In struct {
	// Counts the accounts, and tells nothing.
	DryRun bool
}

type Out struct {
	Accounts int
}

const pageSize = 500

type UseCase struct {
	store         Store
	transactioner cpatomic.Transactioner
	outbox        cpatomic.Outbox
}

var _ Executor = (*UseCase)(nil)

func New(store Store, transactioner cpatomic.Transactioner, outbox cpatomic.Outbox) *UseCase {
	return &UseCase{store: store, transactioner: transactioner, outbox: outbox}
}

// Each account that has done a mission is told what it has done in all, as a fact that finishes a mission tells it, a
// page at a time: each page's events are recorded in a transaction of their own. A mission done meanwhile tells higher
// totals, and a listener keeps the highest.
func (u *UseCase) Execute(ctx context.Context, in In) (Out, error) {
	var (
		out   Out
		after missions.AccountID
	)
	for {
		page, err := u.store.DoneAfter(ctx, after, pageSize)
		if err != nil {
			return out, fmt.Errorf("failed to read a page of the accounts with a mission done: %w", err)
		}
		if !in.DryRun && len(page) > 0 {
			told := make([]proto.Message, 0, len(page))
			for _, done := range page {
				told = append(told, donemessage.Of(done.Account, done.Done))
			}
			if err := u.transactioner.InTx(ctx, func(ctx context.Context) error {
				return u.outbox.Record(ctx, told...)
			}); err != nil {
				return out, fmt.Errorf("failed to record a page of the missions done: %w", err)
			}
		}
		out.Accounts += len(page)

		if len(page) < pageSize {
			return out, nil
		}
		after = page[len(page)-1].Account
	}
}
