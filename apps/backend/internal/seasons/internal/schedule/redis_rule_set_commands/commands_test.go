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
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/calendar"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/schedule"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/schedule/redis_rule_set_commands"
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

	phases := calendar.New(calendar.Config{List: []calendar.Entry{{Number: 0, EndsAt: finale.Add(2 * time.Hour), Finale: 2 * time.Hour}}})
	sched, err := schedule.New(phases.Phases(), schedule.PhaseSets{Days: "default", Finale: "rush", Over: "default"})
	require.NoError(t, err)
	return sched.TurnAt(finale)
}

func TestPlanetAndTheAntibotAreEachToldTheSetInTheirWords(t *testing.T) {
	commands := &sent{}

	require.NoError(t, redis_rule_set_commands.New(commands).Play(t.Context(), rush(t), finale))

	require.Len(t, commands.commands, 2)
	assert.True(t, proto.Equal(&planetv1.PlayRuleSet{RuleSet: "rush", DecidedAt: timestamppb.New(finale)}, commands.commands[0]))
	assert.True(t, proto.Equal(&antibotv1.PlayRuleSet{RuleSet: "rush", DecidedAt: timestamppb.New(finale)}, commands.commands[1]))
}

func TestACommandRedisDidNotTakeIsAnError(t *testing.T) {
	down := errors.New("redis is down")

	err := redis_rule_set_commands.New(&sent{err: down}).Play(t.Context(), rush(t), finale)

	require.ErrorIs(t, err, down)
	assert.Contains(t, err.Error(), `"rush"`)
}
