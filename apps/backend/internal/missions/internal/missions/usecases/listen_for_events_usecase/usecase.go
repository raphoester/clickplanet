package listen_for_events_usecase

import (
	"context"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions/usecases/get_missions_usecase"
)

// Must stay well under Cloudflare's ~125s idle cut, which kills a silent stream.
const DefaultHeartbeat = 30 * time.Second

type Payments interface {
	Subscribe(ctx context.Context, account missions.AccountID) <-chan struct{}
}

type Days interface {
	Execute(ctx context.Context, in get_missions_usecase.In) (get_missions_usecase.Out, error)
}

type Sink interface {
	SendDay(day get_missions_usecase.Out) error
	SendHeartbeat() error
}

func New(payments Payments, days Days, heartbeat time.Duration) *UseCase {
	if heartbeat <= 0 {
		heartbeat = DefaultHeartbeat
	}

	return &UseCase{payments: payments, days: days, heartbeat: heartbeat}
}

type UseCase struct {
	payments  Payments
	days      Days
	heartbeat time.Duration
}

// Subscribed before the first read, so a payment told during it is read again.
func (u *UseCase) Execute(ctx context.Context, account missions.AccountID, sink Sink) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	paid := u.payments.Subscribe(ctx, account)
	if err := u.sendDay(ctx, account, sink); err != nil {
		return err
	}

	heartbeat := time.NewTicker(u.heartbeat)
	defer heartbeat.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil

		case <-heartbeat.C:
			if err := sink.SendHeartbeat(); err != nil {
				return err //nolint:wrapcheck // the stream's own error.
			}

		case _, open := <-paid:
			if !open {
				return nil
			}

			if err := u.sendDay(ctx, account, sink); err != nil {
				return err
			}
		}
	}
}

func (u *UseCase) sendDay(ctx context.Context, account missions.AccountID, sink Sink) error {
	day, err := u.days.Execute(ctx, get_missions_usecase.In{Account: account})
	if err != nil {
		return err //nolint:wrapcheck // the use case named it.
	}

	return sink.SendDay(day) //nolint:wrapcheck // the stream's own error.
}
