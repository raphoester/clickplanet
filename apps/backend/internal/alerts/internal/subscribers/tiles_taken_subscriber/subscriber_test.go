package tiles_taken_subscriber_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/pings"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/subscribers/tiles_taken_subscriber"
)

type heard struct {
	calls [][]pings.Take
}

func (h *heard) Execute(_ context.Context, takes ...pings.Take) error {
	h.calls = append(h.calls, takes)
	return nil
}

var at = time.Date(2026, 11, 2, 18, 0, 0, 0, time.UTC)

func TestATakeFromAnotherFlagIsHeard(t *testing.T) {
	useCase := &heard{}

	err := tiles_taken_subscriber.New(useCase).HandleBatch(t.Context(), []*planetv1.TilesTaken{
		{Country: "gb", TakenAt: timestamppb.New(at), Tiles: []*planetv1.TakenTile{
			{Previous: "ie", Ground: "us", Point: &planetv1.GlobePoint{X: 0.6, Z: 0.8}},
			{Previous: "ie", Ground: ""},
			{Previous: "", Ground: "us"},
			{Previous: "gb", Ground: "us"},
		}},
	})

	require.NoError(t, err)
	assert.Equal(t, [][]pings.Take{{
		{Victim: "ie", Attacker: "gb", Ground: "us", Point: pings.Point{X: 0.6, Z: 0.8}, At: at},
		{Victim: "ie", Attacker: "gb", Ground: "ie", At: at},
	}}, useCase.calls)
}

func TestTheTakesOfABatchAreHeardTogetherInOrder(t *testing.T) {
	useCase := &heard{}

	err := tiles_taken_subscriber.New(useCase).HandleBatch(t.Context(), []*planetv1.TilesTaken{
		{Country: "gb", TakenAt: timestamppb.New(at), Tiles: []*planetv1.TakenTile{{Previous: "ie", Ground: "ie"}}},
		{Country: "fr", TakenAt: timestamppb.New(at), Tiles: []*planetv1.TakenTile{{Previous: "de", Ground: "de"}}},
		{Country: "fr", TakenAt: timestamppb.New(at), Tiles: []*planetv1.TakenTile{{Previous: "fr", Ground: "fr"}}},
	})

	require.NoError(t, err)
	assert.Equal(t, [][]pings.Take{{
		{Victim: "ie", Attacker: "gb", Ground: "ie", At: at},
		{Victim: "de", Attacker: "fr", Ground: "de", At: at},
	}}, useCase.calls, "one call for the batch")
}

func TestABatchThatTookNothingFromAnyoneCallsNothing(t *testing.T) {
	useCase := &heard{}

	err := tiles_taken_subscriber.New(useCase).HandleBatch(t.Context(), []*planetv1.TilesTaken{
		{Country: "gb", TakenAt: timestamppb.New(at), Tiles: []*planetv1.TakenTile{{Ground: "ie"}}},
	})

	require.NoError(t, err)
	assert.Empty(t, useCase.calls)
}
