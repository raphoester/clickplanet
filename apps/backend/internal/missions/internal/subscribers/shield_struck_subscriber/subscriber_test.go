package shield_struck_subscriber_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/flagmissions"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/subscribers/shield_struck_subscriber"
)

type recordingUseCase struct {
	strikes []flagmissions.Strike
}

func (u *recordingUseCase) Execute(_ context.Context, strike flagmissions.Strike) error {
	u.strikes = append(u.strikes, strike)
	return nil
}

var at = time.Date(2026, 10, 28, 12, 0, 0, 0, time.UTC)

func TestAShieldStruckIsAShieldBrokenOfItsOwnerAtItsPlaceInTheLedger(t *testing.T) {
	useCase := &recordingUseCase{}

	require.NoError(t, shield_struck_subscriber.New(useCase).Handle(t.Context(), &planetv1.ShieldStruck{
		Country: "fr", Owner: "dz", StruckAt: timestamppb.New(at), Position: 7, Index: 2,
	}))

	assert.Equal(t, []flagmissions.Strike{{Flag: "fr", Owner: "dz", Position: 7, Index: 2, At: at}}, useCase.strikes)
}

func TestAStrikeWithNoTimeIsRefused(t *testing.T) {
	err := shield_struck_subscriber.New(&recordingUseCase{}).Handle(t.Context(), &planetv1.ShieldStruck{
		Country: "fr", Owner: "dz", Position: 7,
	})

	assert.Error(t, err)
}
