package tile_taken_subscriber_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/record_allegiance_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/subscribers/tile_taken_subscriber"
)

const account = "0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11"

var at = time.Date(2026, 9, 17, 12, 0, 0, 0, time.UTC)

type recordingUseCase struct {
	takes []record_allegiance_usecase.In
}

func (r *recordingUseCase) Execute(_ context.Context, in record_allegiance_usecase.In) error {
	r.takes = append(r.takes, in)
	return nil
}

func TestATakeIsCountedForItsAccountAndItsScopeAtItsTime(t *testing.T) {
	useCase := &recordingUseCase{}

	err := tile_taken_subscriber.New(useCase).Handle(t.Context(), &planetv1.TileTaken{
		AccountId: account, TileId: 42, Country: "fr", TakenAt: timestamppb.New(at), Scope: "2001:db8::/64",
	})

	require.NoError(t, err)
	assert.Equal(t, []record_allegiance_usecase.In{{Account: account, Scope: "2001:db8::/64", Country: "fr", At: at}}, useCase.takes)
}

func TestAnEventWithNoCountryOrNoTimeIsRefused(t *testing.T) {
	useCase := &recordingUseCase{}
	subscriber := tile_taken_subscriber.New(useCase)

	require.Error(t, subscriber.Handle(t.Context(), &planetv1.TileTaken{AccountId: account, TileId: 42, TakenAt: timestamppb.New(at)}))
	require.Error(t, subscriber.Handle(t.Context(), &planetv1.TileTaken{AccountId: account, TileId: 42, Country: "fr"}))
	assert.Empty(t, useCase.takes)
}
