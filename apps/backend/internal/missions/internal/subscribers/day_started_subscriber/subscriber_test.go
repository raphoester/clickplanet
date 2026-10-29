package day_started_subscriber_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	gamemasterv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/gamemaster/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/days"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/subscribers/day_started_subscriber"
)

type recordingUseCase struct {
	days []days.Day
}

func (u *recordingUseCase) Execute(_ context.Context, day days.Day) error {
	u.days = append(u.days, day)
	return nil
}

var (
	starts = time.Date(2026, 10, 28, 21, 0, 0, 0, time.UTC)
	ends   = starts.Add(24 * time.Hour)
)

func TestADayToldIsKeptByItsRoundsID(t *testing.T) {
	useCase := &recordingUseCase{}

	require.NoError(t, day_started_subscriber.New(useCase).Handle(t.Context(), &gamemasterv1.DayStarted{
		Season: 0, Day: 22, RoundId: "season-0/day-22", BookId: "season-0",
		StartsAt: timestamppb.New(starts), EndsAt: timestamppb.New(ends),
	}))

	assert.Equal(t, []days.Day{days.TestingDayAt(starts)}, useCase.days)
}

func TestADayWithNoIDOrNoLengthIsRefused(t *testing.T) {
	for _, event := range []*gamemasterv1.DayStarted{
		{StartsAt: timestamppb.New(starts), EndsAt: timestamppb.New(ends)},
		{RoundId: "season-0/day-22", StartsAt: timestamppb.New(starts), EndsAt: timestamppb.New(starts)},
	} {
		assert.ErrorIs(t, day_started_subscriber.New(&recordingUseCase{}).Handle(t.Context(), event), days.ErrInvalidDay)
	}
}
