package click_tried_subscriber_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/flagmissions"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/subscribers/click_tried_subscriber"
)

type recordingUseCase struct {
	strikes []flagmissions.Strike
}

func (u *recordingUseCase) Execute(_ context.Context, strike flagmissions.Strike) error {
	u.strikes = append(u.strikes, strike)
	return nil
}

var at = time.Date(2026, 10, 28, 12, 0, 0, 0, time.UTC)

func TestAClickAShieldTookIsAShieldBrokenOfTheTilesFlag(t *testing.T) {
	useCase := &recordingUseCase{}
	subscriber := click_tried_subscriber.New(useCase)

	for _, outcome := range []planetv1.ClickOutcome{
		planetv1.ClickOutcome_CLICK_OUTCOME_TAKEN, planetv1.ClickOutcome_CLICK_OUTCOME_STRUCK,
		planetv1.ClickOutcome_CLICK_OUTCOME_THROTTLED, planetv1.ClickOutcome_CLICK_OUTCOME_REFUSED,
	} {
		require.NoError(t, subscriber.Handle(t.Context(), &planetv1.ClickTried{
			Country: "fr", Owner: "dz", TriedAt: timestamppb.New(at), Outcome: outcome,
		}))
	}

	assert.Equal(t, []flagmissions.Strike{{Flag: "fr", Owner: "dz", At: at}}, useCase.strikes)
}

func TestAStrikeWithNoTimeIsRefused(t *testing.T) {
	err := click_tried_subscriber.New(&recordingUseCase{}).Handle(t.Context(), &planetv1.ClickTried{
		Country: "fr", Owner: "dz", Outcome: planetv1.ClickOutcome_CLICK_OUTCOME_STRUCK,
	})

	assert.Error(t, err)
}
