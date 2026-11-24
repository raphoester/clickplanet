package pay_usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/notices"
	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/payments"
)

type Intents interface {
	IntentOf(ctx context.Context, invoice payments.PaymentID) (payments.IntentID, error)
}

type In struct {
	Event   notices.EventID
	Charged payments.Charged
	// An invoice's event does not name its intent: it is read from Stripe.
	Invoice bool
}

func New(
	db cpatomic.Transactioner, inbox notices.Inbox, store payments.Store, intents Intents, outbox cpatomic.Outbox,
) *UseCase {
	return &UseCase{db: db, inbox: inbox, store: store, intents: intents, outbox: outbox}
}

type UseCase struct {
	db      cpatomic.Transactioner
	inbox   notices.Inbox
	store   payments.Store
	intents Intents
	outbox  cpatomic.Outbox
}

// Execute keeps the payment once, and asks the flags module for its time in the same transaction.
func (u *UseCase) Execute(ctx context.Context, in In) error {
	charged := in.Charged
	if in.Invoice {
		intent, err := u.intents.IntentOf(ctx, charged.ID)
		if err != nil {
			return fmt.Errorf("failed to read the invoice's intent: %w", err)
		}
		charged.Intent = intent
	}
	payment, grant, err := payments.Pay(charged)
	if err != nil {
		return err //nolint:wrapcheck // the payment names it.
	}
	if err := u.db.InTx(ctx, func(ctx context.Context) error {
		fresh, err := u.inbox.Keep(ctx, in.Event, charged.At)
		if err != nil || !fresh {
			return err //nolint:wrapcheck // the inbox named it.
		}
		// A session paid by a bank debit is told twice: completed, then async_payment_succeeded.
		if _, err := u.store.Payment(ctx, charged.ID); !errors.Is(err, payments.ErrUnknownPayment) {
			return err //nolint:wrapcheck // the store named it.
		}
		if err := u.store.Save(ctx, payment); err != nil {
			return fmt.Errorf("failed to keep the payment: %w", err)
		}
		if err := u.outbox.Record(ctx, grant); err != nil {
			return fmt.Errorf("failed to ask for the payment's time: %w", err)
		}
		return nil
	}); err != nil {
		return fmt.Errorf("failed to take the payment %s: %w", charged.ID, err)
	}
	return nil
}
