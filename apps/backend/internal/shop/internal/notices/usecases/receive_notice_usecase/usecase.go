package receive_notice_usecase

import (
	"context"

	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/notices"
	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/payments/usecases/pay_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/payments/usecases/take_back_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/renewals/usecases/hear_renewal_usecase"
)

type Pay interface {
	Execute(ctx context.Context, in pay_usecase.In) error
}

type TakeBack interface {
	Execute(ctx context.Context, in take_back_usecase.In) error
}

type HearRenewal interface {
	Execute(ctx context.Context, in hear_renewal_usecase.In) error
}

func New(pay Pay, takeBack TakeBack, hearRenewal HearRenewal) *UseCase {
	return &UseCase{pay: pay, takeBack: takeBack, hearRenewal: hearRenewal}
}

type UseCase struct {
	pay         Pay
	takeBack    TakeBack
	hearRenewal HearRenewal
}

// Execute acts on what a Stripe event told, once per event.
func (u *UseCase) Execute(ctx context.Context, notice notices.Notice) error {
	switch notice.Kind {
	case notices.Paid:
		return u.pay.Execute(ctx, pay_usecase.In{ //nolint:wrapcheck // the use case named it.
			Event: notice.Event, Charged: notice.Payment, Invoice: notice.Invoice,
		})
	case notices.TakenBack:
		return u.takeBack.Execute(ctx, take_back_usecase.In{ //nolint:wrapcheck // the use case named it.
			Event: notice.Event, Intent: notice.Intent, At: notice.At,
		})
	case notices.Renewal:
		return u.hearRenewal.Execute(ctx, hear_renewal_usecase.In{ //nolint:wrapcheck // the use case named it.
			Event: notice.Event, Told: notice.Renewal,
		})
	case notices.Ignored:
		return nil
	}
	return nil
}
