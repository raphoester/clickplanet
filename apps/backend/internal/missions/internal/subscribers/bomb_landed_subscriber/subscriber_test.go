package bomb_landed_subscriber_test

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
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/subscribers/bomb_landed_subscriber"
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

func TestABombReachesTheMissionsWithTheGroundItHit(t *testing.T) {
	useCase := &recordingUseCase{}

	require.NoError(t, bomb_landed_subscriber.New(useCase).Handle(t.Context(), &planetv1.BombLanded{
		AccountId: account, TileId: 42, Ground: "br", LandedAt: timestamppb.New(at), Position: 7,
	}))

	assert.Equal(t, []record_fact_usecase.In{{Account: id, At: at, Fact: missions.BombLanded{Ground: "br", Position: 7}}}, useCase.facts)
}

func TestABombNoAccountDroppedCountsForNobody(t *testing.T) {
	useCase := &recordingUseCase{}

	require.NoError(t, bomb_landed_subscriber.New(useCase).Handle(t.Context(), &planetv1.BombLanded{TileId: 42, LandedAt: timestamppb.New(at)}))

	assert.Empty(t, useCase.facts)
}

func TestABombWithNoTimeIsRefused(t *testing.T) {
	assert.Error(t, bomb_landed_subscriber.New(&recordingUseCase{}).Handle(t.Context(), &planetv1.BombLanded{AccountId: account}))
}
