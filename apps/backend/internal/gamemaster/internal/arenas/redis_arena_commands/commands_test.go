package redis_arena_commands_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/arenas/redis_arena_commands"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments"
)

type sent struct {
	commands []proto.Message
	err      error
}

func (s *sent) Send(_ context.Context, commands ...proto.Message) error {
	if s.err != nil {
		return s.err
	}
	s.commands = append(s.commands, commands...)
	return nil
}

var (
	starts = time.Date(2026, 11, 10, 21, 0, 0, 0, time.UTC)
	ends   = time.Date(2026, 11, 10, 23, 0, 0, 0, time.UTC)
)

const battle = "0192f5a0-7c00-7000-8000-000000000031"

func TestEachCommandIsOneArenaCommandInPlanetsWords(t *testing.T) {
	for _, each := range []struct {
		command moments.ArenaCommand
		wire    *planetv1.ArenaCommand
	}{
		{
			moments.ArenaCommand{Action: moments.OpenArena, Arena: battle, At: starts, EndsAt: ends},
			&planetv1.ArenaCommand{Command: &planetv1.ArenaCommand_OpenArena{OpenArena: &planetv1.OpenArena{
				ArenaId: battle, EndsAt: timestamppb.New(ends),
			}}},
		},
		{
			moments.ArenaCommand{Action: moments.CloseArena, Arena: battle, At: ends, EndsAt: ends},
			&planetv1.ArenaCommand{Command: &planetv1.ArenaCommand_CloseArena{CloseArena: &planetv1.CloseArena{ArenaId: battle}}},
		},
	} {
		commands := &sent{}

		require.NoError(t, redis_arena_commands.New(commands).Send(t.Context(), each.command))

		require.Len(t, commands.commands, 1)
		assert.True(t, proto.Equal(each.wire, commands.commands[0]), "%v", commands.commands[0])
	}
}

func TestACommandRedisDidNotTakeIsAnError(t *testing.T) {
	down := errors.New("redis is down")

	err := redis_arena_commands.New(&sent{err: down}).Send(t.Context(), moments.ArenaCommand{Action: moments.CloseArena, Arena: battle})

	require.ErrorIs(t, err, down)
}
