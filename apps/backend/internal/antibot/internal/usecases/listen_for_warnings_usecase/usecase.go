package listen_for_warnings_usecase

import (
	"context"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/warnings"
)

type Feed interface {
	Subscribe(ctx context.Context, account string) <-chan warnings.Warning
}

type Warnings interface {
	Pending(ctx context.Context, account string) (bool, error)
}

type Sink interface {
	SendWarning() error
	SendHeartbeat() error
}

// Well under Cloudflare's ~125s idle timeout.
const defaultHeartbeat = 30 * time.Second

func New(feed Feed, warnings Warnings, heartbeat time.Duration) *UseCase {
	if heartbeat <= 0 {
		heartbeat = defaultHeartbeat
	}
	return &UseCase{feed: feed, warnings: warnings, heartbeat: heartbeat}
}

type UseCase struct {
	feed      Feed
	warnings  Warnings
	heartbeat time.Duration
}

// Subscribed before the read, so a warning given in between is told, at worst twice.
func (u *UseCase) Execute(ctx context.Context, account string, sink Sink) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	given := u.feed.Subscribe(ctx, account)

	pending, err := u.warnings.Pending(ctx, account)
	if err != nil {
		return err //nolint:wrapcheck // the book already named what failed.
	}
	if err := sink.SendHeartbeat(); err != nil {
		return err //nolint:wrapcheck // the stream's own error.
	}
	if pending {
		if err := sink.SendWarning(); err != nil {
			return err //nolint:wrapcheck // the stream's own error.
		}
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
		case _, open := <-given:
			if !open {
				return nil
			}
			if err := sink.SendWarning(); err != nil {
				return err //nolint:wrapcheck // the stream's own error.
			}
		}
	}
}
