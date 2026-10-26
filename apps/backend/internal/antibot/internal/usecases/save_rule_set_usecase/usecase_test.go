package save_rule_set_usecase_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	antibotv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/antibot/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/guard"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/rules"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/usecases/save_rule_set_usecase"
)

type playing rules.SetName

func (p playing) Plays(name rules.SetName) bool { return name == rules.SetName(p) }

var rush = rules.Named("rush", func(r *antibotv1.Rules) { r.Cohort.CertainMembers = 50 })

func TestASetThatDoesNotPlayIsKeptAsWritten(t *testing.T) {
	sets := rules.NewMemoryStore(rules.Seeded())

	require.NoError(t, save_rule_set_usecase.New(sets, guard.Parts(), playing(rules.Default)).Execute(t.Context(), rush))

	kept, ok := sets.Kept("rush")
	require.True(t, ok)
	assert.True(t, proto.Equal(rush.Rules, kept.Rules))
}

func TestASetThatPlaysNowIsRefused(t *testing.T) {
	sets := rules.NewMemoryStore(rules.Seeded())
	looser := rules.Named(rules.Default, func(r *antibotv1.Rules) { r.Jury.MinSuspects = 3 })

	err := save_rule_set_usecase.New(sets, guard.Parts(), playing(rules.Default)).Execute(t.Context(), looser)

	require.ErrorIs(t, err, save_rule_set_usecase.ErrPlaying, "the rules in force change only through the schedule")
	kept, _ := sets.Kept(rules.Default)
	assert.Equal(t, uint32(2), kept.Rules.GetJury().GetMinSuspects())
}

func TestASetThatCannotPlayIsNotKept(t *testing.T) {
	sets := rules.NewMemoryStore(rules.Seeded())

	err := save_rule_set_usecase.New(sets, guard.Parts(), playing(rules.Default)).Execute(t.Context(),
		rules.Named("rush", func(r *antibotv1.Rules) { r.Cohort.CertainMembers = 1 }))

	require.ErrorIs(t, err, rules.ErrInvalidSet)
	_, ok := sets.Kept("rush")
	assert.False(t, ok)
}

func TestASaveThatFailsSaysSo(t *testing.T) {
	sets := rules.NewMemoryStore()
	down := errors.New("postgres is down")
	sets.FailWith(down)

	err := save_rule_set_usecase.New(sets, guard.Parts(), playing(rules.Default)).Execute(t.Context(), rush)

	require.ErrorIs(t, err, down)
	assert.NotErrorIs(t, err, rules.ErrInvalidSet)
}
