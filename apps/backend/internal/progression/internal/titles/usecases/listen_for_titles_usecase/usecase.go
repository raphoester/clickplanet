package listen_for_titles_usecase

import (
	"context"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/titles"
)

// DefaultHeartbeat stays well under Cloudflare's ~125s idle timeout.
const DefaultHeartbeat = 30 * time.Second

type Feed interface {
	Subscribe(ctx context.Context, account accounts.AccountID) <-chan titles.IDs
}

type Catalog interface {
	Shown(held titles.IDs) []titles.Standing
}

type Sink interface {
	SendTitleEarned(standing titles.Standing) error
	SendHeartbeat() error
}

type UseCase struct {
	feed      Feed
	catalog   Catalog
	heartbeat time.Duration
}

func New(feed Feed, catalog Catalog, heartbeat time.Duration) *UseCase {
	if heartbeat <= 0 {
		heartbeat = DefaultHeartbeat
	}
	return &UseCase{feed: feed, catalog: catalog, heartbeat: heartbeat}
}

// The first heartbeat goes once the stream follows the feed: the page reads it as the stream being live.
func (u *UseCase) Execute(ctx context.Context, account accounts.AccountID, sink Sink) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	earned := u.feed.Subscribe(ctx, account)
	if err := sink.SendHeartbeat(); err != nil {
		return err //nolint:wrapcheck // the stream's own error.
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
		case granted, open := <-earned:
			if !open {
				return nil
			}
			for _, standing := range u.catalog.Shown(granted) {
				if err := sink.SendTitleEarned(standing); err != nil {
					return err //nolint:wrapcheck // the stream's own error.
				}
			}
		}
	}
}
