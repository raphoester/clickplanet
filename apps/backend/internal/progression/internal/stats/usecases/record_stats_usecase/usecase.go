package record_stats_usecase

import (
	"context"
	"fmt"

	"google.golang.org/protobuf/proto"

	progressionv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/progression/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/stats"
)

type Stats interface {
	Record(ctx context.Context, account accounts.AccountID, change func(stats.Stats) stats.Stats) error
}

type Publisher interface {
	Publish(events ...proto.Message)
}

type UseCase struct {
	stats  Stats
	events Publisher
}

func New(stats Stats, events Publisher) *UseCase {
	return &UseCase{stats: stats, events: events}
}

type In struct {
	Account accounts.AccountID
	Change  func(stats.Stats) stats.Stats
}

func (u *UseCase) Execute(ctx context.Context, in In) error {
	if err := u.stats.Record(ctx, in.Account, in.Change); err != nil {
		return fmt.Errorf("failed to record the stats: %w", err)
	}

	u.events.Publish(&progressionv1.StatsChanged{AccountId: in.Account.String()})
	return nil
}
