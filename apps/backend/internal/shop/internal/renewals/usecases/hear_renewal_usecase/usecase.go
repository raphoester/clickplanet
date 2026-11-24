package hear_renewal_usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/notices"
	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/renewals"
)

type In struct {
	Event notices.EventID
	Told  renewals.Told
}

func New(db cpatomic.Transactioner, inbox notices.Inbox, store renewals.Store) *UseCase {
	return &UseCase{db: db, inbox: inbox, store: store}
}

type UseCase struct {
	db    cpatomic.Transactioner
	inbox notices.Inbox
	store renewals.Store
}

// Execute keeps what Stripe told last of a subscription: begun, stopped by its buyer here or on Link, or ended.
func (u *UseCase) Execute(ctx context.Context, in In) error {
	if err := u.db.InTx(ctx, func(ctx context.Context) error {
		fresh, err := u.inbox.Keep(ctx, in.Event, in.Told.At)
		if err != nil || !fresh {
			return err //nolint:wrapcheck // the inbox named it.
		}
		kept, err := u.store.Renewal(ctx, in.Told.ID)
		found := !errors.Is(err, renewals.ErrUnknownRenewal)
		if err != nil && found {
			return fmt.Errorf("failed to read the renewal: %w", err)
		}
		heard, changed, err := renewals.Heard(kept, found, in.Told)
		if err != nil || !changed {
			return err //nolint:wrapcheck // the renewal named it.
		}
		if err := u.store.Save(ctx, heard); err != nil {
			return fmt.Errorf("failed to keep the renewal: %w", err)
		}
		return nil
	}); err != nil {
		return fmt.Errorf("failed to hear the subscription %s: %w", in.Told.ID, err)
	}
	return nil
}
