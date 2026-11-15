package listen_for_warnings_usecase

import (
	"context"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/notices"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/warnings"
)

type Feed[T any] interface {
	Subscribe(ctx context.Context, account string) <-chan T
}

// A warning's book or a notice's.
type Pending interface {
	Pending(ctx context.Context, account string) (bool, error)
}

type Sink interface {
	SendWarning() error
	SendNotice() error
	SendHeartbeat() error
}

// Well under Cloudflare's ~125s idle timeout.
const defaultHeartbeat = 30 * time.Second

func New(
	warningFeed Feed[warnings.Warning],
	warningBook Pending,
	noticeFeed Feed[notices.Notice],
	noticeBook Pending,
	heartbeat time.Duration,
) *UseCase {
	if heartbeat <= 0 {
		heartbeat = defaultHeartbeat
	}
	return &UseCase{
		warningFeed: warningFeed, warnings: warningBook, noticeFeed: noticeFeed, notices: noticeBook, heartbeat: heartbeat,
	}
}

type UseCase struct {
	warningFeed Feed[warnings.Warning]
	warnings    Pending
	noticeFeed  Feed[notices.Notice]
	notices     Pending
	heartbeat   time.Duration
}

// Subscribed before the reads, so a warning or a notice given in between is told, at worst twice.
func (u *UseCase) Execute(ctx context.Context, account string, sink Sink) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	warned := u.warningFeed.Subscribe(ctx, account)
	noticed := u.noticeFeed.Subscribe(ctx, account)

	warning, err := u.warnings.Pending(ctx, account)
	if err != nil {
		return err //nolint:wrapcheck // the book already named what failed.
	}
	notice, err := u.notices.Pending(ctx, account)
	if err != nil {
		return err //nolint:wrapcheck // the book already named what failed.
	}
	if err := sink.SendHeartbeat(); err != nil {
		return err //nolint:wrapcheck // the stream's own error.
	}
	if warning {
		if err := sink.SendWarning(); err != nil {
			return err //nolint:wrapcheck // the stream's own error.
		}
	}
	if notice {
		if err := sink.SendNotice(); err != nil {
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
		case _, open := <-warned:
			if !open {
				return nil
			}
			if err := sink.SendWarning(); err != nil {
				return err //nolint:wrapcheck // the stream's own error.
			}
		case _, open := <-noticed:
			if !open {
				return nil
			}
			if err := sink.SendNotice(); err != nil {
				return err //nolint:wrapcheck // the stream's own error.
			}
		}
	}
}
