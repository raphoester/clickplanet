package save_rule_set_usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	antibotv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/antibot/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/guard"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/rules"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/usecases/play_rules_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/usecases/save_rule_set_usecase"
)

type players struct {
	plays int
	err   error
}

func (p *players) Execute(context.Context) (play_rules_usecase.Taken, bool, error) {
	p.plays++
	return play_rules_usecase.Taken{}, false, p.err
}

var rush = rules.Named("rush", func(r *antibotv1.Rules) { r.Cohort.CertainMembers = 50 })

func TestASetIsKeptAsWrittenThenTheRulesArePlayedAgain(t *testing.T) {
	sets, played := rules.NewMemoryStore(rules.Seeded()), &players{}

	require.NoError(t, save_rule_set_usecase.New(sets, guard.Parts(), played).Execute(t.Context(), rush))

	kept, ok := sets.Kept("rush")
	require.True(t, ok)
	assert.True(t, proto.Equal(rush.Rules, kept.Rules))
	assert.Equal(t, 1, played.plays, "the set saved may be the one playing")
}

func TestASetThatCannotPlayIsNotKept(t *testing.T) {
	sets, played := rules.NewMemoryStore(rules.Seeded()), &players{}

	err := save_rule_set_usecase.New(sets, guard.Parts(), played).Execute(t.Context(),
		rules.Named("rush", func(r *antibotv1.Rules) { r.Cohort.CertainMembers = 1 }))

	require.ErrorIs(t, err, rules.ErrInvalidSet)
	_, ok := sets.Kept("rush")
	assert.False(t, ok)
	assert.Zero(t, played.plays)
}

func TestASaveThatFailsSaysSo(t *testing.T) {
	sets, played := rules.NewMemoryStore(), &players{}
	down := errors.New("postgres is down")
	sets.FailWith(down)

	err := save_rule_set_usecase.New(sets, guard.Parts(), played).Execute(t.Context(), rush)

	require.ErrorIs(t, err, down)
	assert.NotErrorIs(t, err, rules.ErrInvalidSet)
	assert.Zero(t, played.plays)
}

func TestASetKeptThatCouldNotBePlayedSaysSo(t *testing.T) {
	sets, played := rules.NewMemoryStore(rules.Seeded()), &players{err: errors.New("redis is down")}

	err := save_rule_set_usecase.New(sets, guard.Parts(), played).Execute(t.Context(), rush)

	require.ErrorContains(t, err, `kept the rule set "rush", but could not play the rules`)
	_, ok := sets.Kept("rush")
	assert.True(t, ok)
}
