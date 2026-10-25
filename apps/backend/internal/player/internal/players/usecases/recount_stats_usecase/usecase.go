package recount_stats_usecase

import (
	"context"
	"fmt"

	"google.golang.org/protobuf/proto"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
)

type Stats interface {
	StatsAfter(ctx context.Context, after players.AccountID, limit int) ([]players.Stats, error)
	Record(ctx context.Context, account players.AccountID, change func(players.Stats) players.Stats) error
}

type Ledger interface {
	Recounts(ctx context.Context, accounts []players.AccountID) (map[players.AccountID]players.Recount, error)
}

type Publisher interface {
	Publish(event proto.Message)
}

type Executor interface {
	Execute(ctx context.Context) (int, error)
}

const pageSize = 500

type UseCase struct {
	stats  Stats
	ledger Ledger
	events Publisher
}

var _ Executor = (*UseCase)(nil)

func New(stats Stats, ledger Ledger, events Publisher) *UseCase {
	return &UseCase{stats: stats, ledger: ledger, events: events}
}

func (u *UseCase) Execute(ctx context.Context) (int, error) {
	var (
		recounted int
		after     players.AccountID
	)
	for {
		page, err := u.stats.StatsAfter(ctx, after, pageSize)
		if err != nil {
			return recounted, fmt.Errorf("failed to read a page of stats: %w", err)
		}

		recounts, err := u.ledger.Recounts(ctx, players.AccountsOf(page))
		if err != nil {
			return recounted, fmt.Errorf("failed to ask planet's ledger about a page of accounts: %w", err)
		}

		for _, stats := range page {
			recount := recounts[stats.Account()]
			if stats.Recounted(recount) == stats {
				continue
			}
			change := func(stats players.Stats) players.Stats { return stats.Recounted(recount) }
			if err := u.stats.Record(ctx, stats.Account(), change); err != nil {
				return recounted, fmt.Errorf("failed to recount the stats of %s: %w", stats.Account(), err)
			}
			u.events.Publish(&playerv1.StatsChanged{AccountId: stats.Account().String()})
			recounted++
		}

		if len(page) < pageSize {
			return recounted, nil
		}
		after = page[len(page)-1].Account()
	}
}
