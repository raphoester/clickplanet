package flag_played_subscriber_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/catalog/usecases/keep_played_flag_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/subscribers/flag_played_subscriber"
)

var takenAt = time.Date(2026, 11, 23, 9, 0, 0, 0, time.UTC)

type useCase struct {
	ins []keep_played_flag_usecase.In
}

func (u *useCase) Execute(_ context.Context, in keep_played_flag_usecase.In) error {
	u.ins = append(u.ins, in)
	return nil
}

func taken(flag string, at time.Time) *planetv1.TilesTaken {
	return &planetv1.TilesTaken{Country: flag, TakenAt: timestamppb.New(at)}
}

func TestEachFlagOpenedSinceThatABatchTookForIsAskedForOnceAtItsFirstTake(t *testing.T) {
	keeping := &useCase{}

	require.NoError(t, flag_played_subscriber.NewTaken(keeping).HandleBatch(t.Context(), []*planetv1.TilesTaken{
		taken("fr", takenAt), taken("r-k3p9x2ma0b", takenAt.Add(time.Second)), taken("gb-eng", takenAt),
		taken("r-k3p9x2ma0b", takenAt.Add(2*time.Second)), taken("r-0000000001", takenAt.Add(3*time.Second)),
	}))

	assert.Equal(t, []keep_played_flag_usecase.In{
		{Flag: "r-k3p9x2ma0b", PlayedAt: takenAt.Add(time.Second)}, {Flag: "r-0000000001", PlayedAt: takenAt.Add(3 * time.Second)},
	}, keeping.ins)
}

func TestAPledgeOfAFlagOpenedSinceIsAskedFor(t *testing.T) {
	keeping := &useCase{}
	pledged := flag_played_subscriber.NewPledged(keeping)

	require.NoError(t, pledged.Handle(t.Context(), &planetv1.AllegiancePledged{Country: "fr", PledgedAt: timestamppb.New(takenAt)}))
	require.NoError(t, pledged.Handle(t.Context(),
		&planetv1.AllegiancePledged{Country: "r-k3p9x2ma0b", PledgedAt: timestamppb.New(takenAt)}))

	assert.Equal(t, []keep_played_flag_usecase.In{{Flag: "r-k3p9x2ma0b", PlayedAt: takenAt}}, keeping.ins)
}
