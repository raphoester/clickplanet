package listen_for_events_usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/levels"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/stats"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/titles"
)

// DefaultHeartbeat stays well under Cloudflare's ~125s idle timeout.
const DefaultHeartbeat = 30 * time.Second

type TitleFeed interface {
	Subscribe(ctx context.Context, account accounts.AccountID) <-chan titles.IDs
}

type Catalog interface {
	Shown(held titles.IDs) []titles.Standing
}

// LevelFeed follows the account from what its stats gave as the stream opened, to tell the XP each fact gives past it.
type LevelFeed interface {
	Subscribe(ctx context.Context, account accounts.AccountID, earned levels.Earned) <-chan levels.Told
}

type Stats interface {
	Stats(ctx context.Context, account accounts.AccountID) (stats.Stats, error)
}

type Sink interface {
	SendTitleEarned(standing titles.Standing) error
	SendLevelReached(standing levels.Standing) error
	SendXPGained(gained levels.Gained) error
	SendHeartbeat() error
}

type UseCase struct {
	titles    TitleFeed
	catalog   Catalog
	levels    LevelFeed
	stats     Stats
	heartbeat time.Duration
}

func New(titles TitleFeed, catalog Catalog, levels LevelFeed, stats Stats, heartbeat time.Duration) *UseCase {
	if heartbeat <= 0 {
		heartbeat = DefaultHeartbeat
	}
	return &UseCase{titles: titles, catalog: catalog, levels: levels, stats: stats, heartbeat: heartbeat}
}

// The first heartbeat goes once the stream follows both feeds: the page reads it as the stream being live. A stream
// that cannot read the account's stats fails: from no start, the first gain told would be the account's whole XP.
func (u *UseCase) Execute(ctx context.Context, account accounts.AccountID, sink Sink) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	counted, err := u.stats.Stats(ctx, account)
	var earned levels.Earned
	switch {
	case errors.Is(err, stats.ErrNoStats):
	case err != nil:
		return fmt.Errorf("failed to read the stats: %w", err)
	default:
		earned = levels.Scale{}.EarnedOf(counted)
	}

	titled := u.titles.Subscribe(ctx, account)
	told := u.levels.Subscribe(ctx, account, earned)
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
		case granted, open := <-titled:
			if !open {
				return nil
			}
			for _, standing := range u.catalog.Shown(granted) {
				if err := sink.SendTitleEarned(standing); err != nil {
					return err //nolint:wrapcheck // the stream's own error.
				}
			}
		case level, open := <-told:
			if !open {
				return nil
			}
			if err := send(sink, level); err != nil {
				return err //nolint:wrapcheck // the stream's own error.
			}
		}
	}
}

func send(sink Sink, told levels.Told) error {
	switch told := told.(type) {
	case levels.Reached:
		return sink.SendLevelReached(told.Standing) //nolint:wrapcheck // the stream's own error.
	case levels.Gained:
		return sink.SendXPGained(told) //nolint:wrapcheck // the stream's own error.
	default:
		return nil
	}
}
