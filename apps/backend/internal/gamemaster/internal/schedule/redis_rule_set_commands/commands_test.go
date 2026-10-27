package redis_rule_set_commands_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	antibotv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/antibot/v1"
	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	scoringv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/scoring/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/schedule"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/schedule/redis_rule_set_commands"
)

type sent struct {
	commands []proto.Message
	err      error
}

func (s *sent) Send(_ context.Context, commands ...proto.Message) error {
	s.commands = append(s.commands, commands...)
	return s.err
}

var finale = time.Date(2026, 10, 31, 21, 0, 0, 0, time.UTC)

func rush(t *testing.T) schedule.Turn {
	t.Helper()

	return schedule.New(schedule.EntrySnapshot{
		Set: "rush", StartsAt: finale, EndsAt: finale.Add(2 * time.Hour), Priority: 1, MadeAt: finale.Add(-time.Hour),
	}.Restore()).TurnAt(finale)
}

func TestPlanetTheAntibotAndScoringAreEachToldTheSetInTheirWords(t *testing.T) {
	commands := &sent{}

	require.NoError(t, redis_rule_set_commands.New(commands).Play(t.Context(), rush(t), finale))

	require.Len(t, commands.commands, 3)
	assert.True(t, proto.Equal(&planetv1.PlayRuleSet{RuleSet: "rush", DecidedAt: timestamppb.New(finale)}, commands.commands[0]))
	assert.True(t, proto.Equal(&antibotv1.PlayRuleSet{RuleSet: "rush", DecidedAt: timestamppb.New(finale)}, commands.commands[1]))
	assert.True(t, proto.Equal(&scoringv1.PlayRuleSet{
		RuleSet: "rush", DecidedAt: timestamppb.New(finale),
		StartsAt: timestamppb.New(finale), EndsAt: timestamppb.New(finale.Add(2 * time.Hour)),
	}, commands.commands[2]), "scoring is told the entry's window: %v", commands.commands[2])
}

func TestDefaultIsToldToScoringWithNoWindow(t *testing.T) {
	commands := &sent{}

	require.NoError(t, redis_rule_set_commands.New(commands).Play(t.Context(), schedule.New().TurnAt(finale), finale))

	require.Len(t, commands.commands, 3)
	assert.True(t, proto.Equal(&scoringv1.PlayRuleSet{RuleSet: "default", DecidedAt: timestamppb.New(finale)}, commands.commands[2]))
}

func TestACommandRedisDidNotTakeIsAnError(t *testing.T) {
	down := errors.New("redis is down")

	err := redis_rule_set_commands.New(&sent{err: down}).Play(t.Context(), rush(t), finale)

	require.ErrorIs(t, err, down)
	assert.Contains(t, err.Error(), `"rush"`)
}
