package tile_taken_subscriber_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/pings"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/subscribers/tile_taken_subscriber"
)

type heard struct {
	takes []pings.Take
}

func (h *heard) Execute(_ context.Context, take pings.Take) error {
	h.takes = append(h.takes, take)
	return nil
}

func TestATakeFromAnotherFlagIsHeard(t *testing.T) {
	at := time.Date(2026, 11, 2, 18, 0, 0, 0, time.UTC)
	useCase := &heard{}
	subscriber := tile_taken_subscriber.New(useCase)

	for _, event := range []*planetv1.TileTaken{
		{Country: "gb", Previous: "ie", Ground: "us", TakenAt: timestamppb.New(at), Point: &planetv1.GlobePoint{X: 0.6, Z: 0.8}},
		{Country: "gb", Previous: "ie", Ground: "", TakenAt: timestamppb.New(at)},
		{Country: "gb", Previous: "", Ground: "us", TakenAt: timestamppb.New(at)},
		{Country: "gb", Previous: "gb", Ground: "us", TakenAt: timestamppb.New(at)},
	} {
		require.NoError(t, subscriber.Handle(t.Context(), event))
	}

	assert.Equal(t, []pings.Take{
		{Victim: "ie", Attacker: "gb", Ground: "us", Point: pings.Point{X: 0.6, Z: 0.8}, At: at},
		{Victim: "ie", Attacker: "gb", Ground: "ie", At: at},
	}, useCase.takes)
}
