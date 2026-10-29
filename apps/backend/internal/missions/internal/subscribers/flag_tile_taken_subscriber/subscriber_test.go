package flag_tile_taken_subscriber_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/flagmissions"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/subscribers/flag_tile_taken_subscriber"
)

type recordingUseCase struct {
	takes []flagmissions.Take
}

func (u *recordingUseCase) Execute(_ context.Context, take flagmissions.Take) error {
	u.takes = append(u.takes, take)
	return nil
}

var at = time.Date(2026, 10, 28, 12, 0, 0, 0, time.UTC)

func TestATakeCountsForTheFlagItWasTakenFor(t *testing.T) {
	useCase := &recordingUseCase{}

	require.NoError(t, flag_tile_taken_subscriber.New(useCase).Handle(t.Context(), &planetv1.TileTaken{
		Country: "fr", Previous: "dz", Ground: "br", Position: 7, Index: 2, TakenAt: timestamppb.New(at),
	}))

	assert.Equal(t, []flagmissions.Take{{Flag: "fr", Ground: "br", Position: 7, Index: 2, At: at}}, useCase.takes)
}

func TestATakeWithNoTimeIsRefused(t *testing.T) {
	err := flag_tile_taken_subscriber.New(&recordingUseCase{}).Handle(t.Context(), &planetv1.TileTaken{Country: "fr"})

	assert.Error(t, err)
}
