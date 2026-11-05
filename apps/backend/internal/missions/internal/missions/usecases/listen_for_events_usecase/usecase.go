package listen_for_events_usecase

import (
	"context"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions/usecases/get_missions_usecase"
)

// Must stay well under Cloudflare's ~125s idle cut, which kills a silent stream.
const DefaultHeartbeat = 30 * time.Second

const DefaultGap = time.Second

type Changes interface {
	Subscribe(ctx context.Context, account missions.AccountID) <-chan struct{}
}

type Days interface {
	Execute(ctx context.Context, in get_missions_usecase.In) (get_missions_usecase.Out, error)
}

type Sink interface {
	SendDay(day get_missions_usecase.Out) error
	SendHeartbeat() error
}

func New(changes Changes, days Days, heartbeat, gap time.Duration) *UseCase {
	if heartbeat <= 0 {
		heartbeat = DefaultHeartbeat
	}

	return &UseCase{changes: changes, days: days, heartbeat: heartbeat, gap: gap}
}

type UseCase struct {
	changes   Changes
	days      Days
	heartbeat time.Duration
	gap       time.Duration
}

// Subscribed before the first read, so a change told during it is read again.
func (u *UseCase) Execute(ctx context.Context, account missions.AccountID, sink Sink) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	changed := u.changes.Subscribe(ctx, account)
	if err := u.sendDay(ctx, account, sink); err != nil {
		return err
	}
	sent := time.Now()

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

		case _, open := <-changed:
			if !open || !waitUntil(ctx, sent.Add(u.gap)) {
				return nil
			}
			// A change told during the wait is in the day read now.
			select {
			case <-changed:
			default:
			}

			if err := u.sendDay(ctx, account, sink); err != nil {
				return err
			}
			sent = time.Now()
		}
	}
}

func waitUntil(ctx context.Context, at time.Time) bool {
	timer := time.NewTimer(time.Until(at))
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

func (u *UseCase) sendDay(ctx context.Context, account missions.AccountID, sink Sink) error {
	day, err := u.days.Execute(ctx, get_missions_usecase.In{Account: account})
	if err != nil {
		return err //nolint:wrapcheck // the use case named it.
	}

	return sink.SendDay(day) //nolint:wrapcheck // the stream's own error.
}
