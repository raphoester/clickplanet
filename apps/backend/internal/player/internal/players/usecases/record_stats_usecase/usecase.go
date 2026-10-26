package record_stats_usecase

import (
	"context"
	"fmt"

	"google.golang.org/protobuf/proto"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
)

type Stats interface {
	Record(ctx context.Context, account players.AccountID, change func(players.Stats) players.Stats) error
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
	Account players.AccountID
	Change  func(players.Stats) players.Stats
}

func (u *UseCase) Execute(ctx context.Context, in In) error {
	if err := u.stats.Record(ctx, in.Account, in.Change); err != nil {
		return fmt.Errorf("failed to record the stats: %w", err)
	}

	u.events.Publish(&playerv1.StatsChanged{AccountId: in.Account.String()})
	return nil
}
