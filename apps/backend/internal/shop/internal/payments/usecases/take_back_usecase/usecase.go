package take_back_usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/notices"
	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/payments"
)

type In struct {
	Event  notices.EventID
	Intent payments.IntentID
	At     time.Time
}

func New(db cpatomic.Transactioner, inbox notices.Inbox, store payments.Store, outbox cpatomic.Outbox) *UseCase {
	return &UseCase{db: db, inbox: inbox, store: store, outbox: outbox}
}

type UseCase struct {
	db     cpatomic.Transactioner
	inbox  notices.Inbox
	store  payments.Store
	outbox cpatomic.Outbox
}

// Execute takes a payment's time back once: a refund in full or a dispute lost. One of the account's other sales is
// none of the shop's.
func (u *UseCase) Execute(ctx context.Context, in In) error {
	if err := u.db.InTx(ctx, func(ctx context.Context) error {
		fresh, err := u.inbox.Keep(ctx, in.Event, in.At)
		if err != nil || !fresh {
			return err //nolint:wrapcheck // the inbox named it.
		}
		payment, err := u.store.OfIntent(ctx, in.Intent)
		if errors.Is(err, payments.ErrUnknownPayment) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("failed to read the payment: %w", err)
		}
		taken, command, changed := payment.TakenBack(in.At)
		if !changed {
			return nil
		}
		if err := u.store.Save(ctx, taken); err != nil {
			return fmt.Errorf("failed to keep the payment taken back: %w", err)
		}
		if err := u.outbox.Record(ctx, command); err != nil {
			return fmt.Errorf("failed to ask for the payment's time back: %w", err)
		}
		return nil
	}); err != nil {
		return fmt.Errorf("failed to take back the payment of %s: %w", in.Intent, err)
	}
	return nil
}
