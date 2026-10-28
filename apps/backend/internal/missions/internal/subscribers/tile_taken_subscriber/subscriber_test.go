package tile_taken_subscriber_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/subscribers/tile_taken_subscriber"
)

type recordingUseCase struct {
	takes []missions.Take
}

func (u *recordingUseCase) Execute(_ context.Context, take missions.Take) error {
	u.takes = append(u.takes, take)
	return nil
}

const account = "0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11"

var at = time.Date(2026, 10, 26, 12, 0, 0, 0, time.UTC)

func TestATakeReachesTheMissionsWithWhoseTileItWasAndWhoseGround(t *testing.T) {
	useCase := &recordingUseCase{}

	require.NoError(t, tile_taken_subscriber.New(useCase).Handle(t.Context(), &planetv1.TileTaken{
		AccountId: account, TileId: 42, Country: "fr", Previous: "dz", Ground: "br", TakenAt: timestamppb.New(at), Position: 7,
		Index: 3,
	}))

	id, err := missions.AccountIDOf(account)
	require.NoError(t, err)
	assert.Equal(t, []missions.Take{{Account: id, Flag: "fr", Previous: "dz", Ground: "br", At: at, Position: 7, Index: 3}}, useCase.takes)
}

func TestAnEventWithNoAccountOrNoTimeIsRefused(t *testing.T) {
	subscriber := tile_taken_subscriber.New(&recordingUseCase{})

	require.ErrorIs(t, subscriber.Handle(t.Context(), &planetv1.TileTaken{TileId: 42, TakenAt: timestamppb.New(at)}),
		missions.ErrInvalidAccount)
	assert.Error(t, subscriber.Handle(t.Context(), &planetv1.TileTaken{AccountId: account, TileId: 42}))
}
