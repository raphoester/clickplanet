package redis_season_commands_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	scoringv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/scoring/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/season"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/season/redis_season_commands"
)

type sent struct {
	commands []proto.Message
	err      error
}

func (s *sent) Send(_ context.Context, commands ...proto.Message) error {
	s.commands = append(s.commands, commands...)
	return s.err
}

var (
	starts = time.Date(2026, 10, 3, 0, 0, 0, 0, time.UTC)
	ends   = time.Date(2026, 10, 31, 23, 0, 0, 0, time.UTC)
	now    = time.Date(2026, 10, 28, 12, 0, 0, 0, time.UTC)
	zero   = season.Snapshot{Number: 0, StartsAt: starts, EndsAt: ends}.Restore()
)

func TestScoringIsToldTheSeasonsWindowInItsWords(t *testing.T) {
	commands := &sent{}

	require.NoError(t, redis_season_commands.New(commands).Play(t.Context(), zero, now))

	require.Len(t, commands.commands, 1)
	assert.True(t, proto.Equal(&scoringv1.PlaySeason{
		Number: 0, StartsAt: timestamppb.New(starts), EndsAt: timestamppb.New(ends), DecidedAt: timestamppb.New(now),
	}, commands.commands[0]), "%v", commands.commands[0])
}

func TestACommandRedisDidNotTakeIsAnError(t *testing.T) {
	down := errors.New("redis is down")

	err := redis_season_commands.New(&sent{err: down}).Play(t.Context(), zero, now)

	require.ErrorIs(t, err, down)
	assert.Contains(t, err.Error(), "season 0")
}
