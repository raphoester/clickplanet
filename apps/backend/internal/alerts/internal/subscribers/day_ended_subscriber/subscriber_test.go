package day_ended_subscriber_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	gamemasterv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/gamemaster/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/pings"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/subscribers/day_ended_subscriber"
)

type heard struct {
	days []pings.EndedDay
}

func (h *heard) Execute(_ context.Context, day pings.EndedDay) error {
	h.days = append(h.days, day)
	return nil
}

func TestADayIsToldByItsRound(t *testing.T) {
	ended := time.Date(2026, 11, 3, 21, 0, 0, 0, time.UTC)
	useCase := &heard{}

	require.NoError(t, day_ended_subscriber.New(useCase).Handle(t.Context(), &gamemasterv1.DayEnded{
		Season: 1, Day: 3, RoundId: "season-1/day-3", EndedAt: timestamppb.New(ended),
	}))

	assert.Equal(t, []pings.EndedDay{{Round: "season-1/day-3", Season: 1, Number: 3, EndedAt: ended}}, useCase.days)
}

func TestADayWithNoEndOrNoRoundIsRefused(t *testing.T) {
	useCase := &heard{}
	subscriber := day_ended_subscriber.New(useCase)

	require.Error(t, subscriber.Handle(t.Context(), &gamemasterv1.DayEnded{Season: 1, Day: 3, RoundId: "season-1/day-3"}))
	require.Error(t, subscriber.Handle(t.Context(), &gamemasterv1.DayEnded{Season: 1, Day: 3, EndedAt: timestamppb.Now()}))

	assert.Empty(t, useCase.days)
}
