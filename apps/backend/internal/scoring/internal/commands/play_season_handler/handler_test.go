package play_season_handler_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	scoringv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/scoring/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/calendar"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/commands/play_season_handler"
)

type stubUseCase struct {
	asked []calendar.Decision
	err   error
}

func (s *stubUseCase) Execute(_ context.Context, decision calendar.Decision) (bool, error) {
	s.asked = append(s.asked, decision)
	return s.err == nil, s.err
}

var (
	starts  = time.Date(2026, 10, 3, 0, 0, 0, 0, time.UTC)
	ends    = time.Date(2026, 10, 31, 23, 0, 0, 0, time.UTC)
	decided = time.Date(2026, 10, 28, 12, 0, 0, 0, time.UTC)
	command = &scoringv1.PlaySeason{
		Number: 1, StartsAt: timestamppb.New(starts), EndsAt: timestamppb.New(ends), DecidedAt: timestamppb.New(decided),
	}
)

func TestTheSeasonToldIsPlayedInScoringsWords(t *testing.T) {
	useCase := &stubUseCase{}

	require.NoError(t, play_season_handler.New(useCase).Handle(t.Context(), command))

	assert.Equal(t, []calendar.Decision{{Window: calendar.Window{Number: 1, StartsAt: starts, EndsAt: ends}, DecidedAt: decided}}, useCase.asked)
}

func TestAnInvalidWindowIsDropped(t *testing.T) {
	require.NoError(t, play_season_handler.New(&stubUseCase{err: calendar.ErrInvalidSeason}).Handle(t.Context(), command))
}

func TestAFailureIsDeliveredAgain(t *testing.T) {
	down := errors.New("postgres is down")

	require.ErrorIs(t, play_season_handler.New(&stubUseCase{err: down}).Handle(t.Context(), command), down)
}
