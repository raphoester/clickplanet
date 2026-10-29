package shield_placed_subscriber_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions/usecases/record_fact_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/subscribers/shield_placed_subscriber"
)

type recordingUseCase struct {
	facts []record_fact_usecase.In
}

func (u *recordingUseCase) Execute(_ context.Context, in record_fact_usecase.In) error {
	u.facts = append(u.facts, in)
	return nil
}

const account = "0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11"

var (
	at    = time.Date(2026, 10, 26, 12, 0, 0, 0, time.UTC)
	id, _ = missions.AccountIDOf(account)
)

func TestAShieldReachesTheMissionsWithTheShieldsItsTileHolds(t *testing.T) {
	useCase := &recordingUseCase{}

	require.NoError(t, shield_placed_subscriber.New(useCase).Handle(t.Context(), &planetv1.ShieldPlaced{
		AccountId: account, TileId: 42, Country: "fr", Shields: 10, PlacedAt: timestamppb.New(at), Position: 7,
	}))

	assert.Equal(t, []record_fact_usecase.In{{Account: id, At: at, Fact: missions.ShieldPlaced{Shields: 10, Position: 7}}}, useCase.facts)
}

func TestAShieldNoAccountPlacedCountsForNobody(t *testing.T) {
	useCase := &recordingUseCase{}

	require.NoError(t, shield_placed_subscriber.New(useCase).Handle(t.Context(), &planetv1.ShieldPlaced{TileId: 42, PlacedAt: timestamppb.New(at)}))

	assert.Empty(t, useCase.facts)
}

func TestAShieldWithNoTimeIsRefused(t *testing.T) {
	assert.Error(t, shield_placed_subscriber.New(&recordingUseCase{}).Handle(t.Context(), &planetv1.ShieldPlaced{AccountId: account}))
}
