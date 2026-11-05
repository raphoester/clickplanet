package battle_started_subscriber_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	gamemasterv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/gamemaster/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/pings"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/subscribers/battle_started_subscriber"
)

type heard struct {
	starts []pings.Start
}

func (h *heard) Execute(_ context.Context, start pings.Start) error {
	h.starts = append(h.starts, start)
	return nil
}

func TestABattleAndAFinaleAreToldByTheirRound(t *testing.T) {
	starts := time.Date(2026, 11, 2, 18, 0, 0, 0, time.UTC)
	ends := starts.Add(2 * time.Hour)
	useCase := &heard{}
	subscriber := battle_started_subscriber.New(useCase)

	require.NoError(t, subscriber.Handle(t.Context(), &gamemasterv1.BattleStarted{
		Season: 1, Number: 2, EntryId: "entry", Times: 2, RoundId: "season-1/battle-2",
		StartsAt: timestamppb.New(starts), EndsAt: timestamppb.New(ends),
	}))
	require.NoError(t, subscriber.Handle(t.Context(), &gamemasterv1.BattleStarted{
		Season: 1, Number: 3, Times: 3, EndsSeason: true, RoundId: "season-1/finale",
		StartsAt: timestamppb.New(starts), EndsAt: timestamppb.New(ends),
	}))

	assert.Equal(t, []pings.Start{
		{Key: "season-1/battle-2", Kind: pings.BattleStart, Season: 1, Times: 2, StartsAt: starts, EndsAt: ends},
		{Key: "season-1/finale", Kind: pings.FinaleStart, Season: 1, Times: 3, StartsAt: starts, EndsAt: ends},
	}, useCase.starts)
}
