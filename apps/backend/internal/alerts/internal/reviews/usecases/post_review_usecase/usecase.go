package post_review_usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/reviews"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Desk interface {
	Tell(ctx context.Context, review reviews.Review) error
}

const timeout = 10 * time.Second

func New(clock cptime.Clock, store reviews.Store, desk Desk, transactioner cpatomic.Transactioner) *UseCase {
	return &UseCase{clock: clock, store: store, desk: desk, transactioner: transactioner}
}

type UseCase struct {
	clock         cptime.Clock
	store         reviews.Store
	desk          Desk
	transactioner cpatomic.Transactioner
}

// Told once, in the transaction of the post: a post Discord refused keeps nothing, and the event comes again.
func (u *UseCase) Execute(ctx context.Context, review reviews.Review) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	told, err := u.store.Told(ctx, review.Submission, review.Cause)
	if err != nil {
		return fmt.Errorf("failed to read whether %s was told: %w", review.Submission, err)
	}
	if told {
		return nil
	}
	return u.transactioner.InTx(ctx, func(ctx context.Context) error { //nolint:wrapcheck // each step named its own.
		if err := u.store.SaveTold(ctx, review.Submission, review.Cause, u.clock.Now()); err != nil {
			return fmt.Errorf("failed to keep the post: %w", err)
		}
		return u.desk.Tell(ctx, review)
	})
}
