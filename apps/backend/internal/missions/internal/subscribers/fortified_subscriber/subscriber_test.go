package fortified_subscriber_test

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
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/subscribers/fortified_subscriber"
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

func TestAFortifyReachesTheMissionsWithItsTiles(t *testing.T) {
	useCase := &recordingUseCase{}

	require.NoError(t, fortified_subscriber.New(useCase).Handle(t.Context(), &planetv1.Fortified{
		AccountId: account, LandmassId: 3, Tiles: 60, FortifiedAt: timestamppb.New(at), Position: 7,
	}))

	assert.Equal(t, []record_fact_usecase.In{{Account: id, At: at, Fact: missions.Fortified{Tiles: 60, Position: 7}}}, useCase.facts)
}

func TestAFortifyNoAccountEarnedCountsForNobody(t *testing.T) {
	useCase := &recordingUseCase{}

	require.NoError(t, fortified_subscriber.New(useCase).Handle(t.Context(), &planetv1.Fortified{LandmassId: 3, FortifiedAt: timestamppb.New(at)}))

	assert.Empty(t, useCase.facts)
}

func TestAFortifyWithNoTimeIsRefused(t *testing.T) {
	assert.Error(t, fortified_subscriber.New(&recordingUseCase{}).Handle(t.Context(), &planetv1.Fortified{AccountId: account}))
}
