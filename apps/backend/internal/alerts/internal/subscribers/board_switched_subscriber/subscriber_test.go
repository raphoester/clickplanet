package board_switched_subscriber_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/pings"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/subscribers/board_switched_subscriber"
)

type told struct {
	starts []pings.Start
}

func (t *told) Execute(_ context.Context, start pings.Start) error {
	t.starts = append(t.starts, start)
	return nil
}

var at = time.Date(2026, 11, 22, 4, 0, 0, 0, time.UTC)

func TestTheWorldsSwitchIsToldAsAStartOfTheWorldsNewBoard(t *testing.T) {
	useCase := &told{}

	require.NoError(t, board_switched_subscriber.New(useCase).Handle(t.Context(), &planetv1.BoardSwitched{
		Map: "world", From: "earth-1", To: "earth-2", FromTiles: 260503, ToTiles: 1004682, SwitchedAt: timestamppb.New(at),
	}))

	assert.Equal(t, []pings.Start{{Key: "world/earth-2", Kind: pings.WorldGrew, Times: 4, StartsAt: at, EndsAt: at}},
		useCase.starts, "told once by its board, as a season by its book")
}

func TestAnotherMapsSwitchIsNotTold(t *testing.T) {
	useCase := &told{}

	require.NoError(t, board_switched_subscriber.New(useCase).Handle(t.Context(), &planetv1.BoardSwitched{
		Map: "battle", From: "earth-1", To: "siege-1", FromTiles: 260503, ToTiles: 27259, SwitchedAt: timestamppb.New(at),
	}))

	assert.Empty(t, useCase.starts)
}
