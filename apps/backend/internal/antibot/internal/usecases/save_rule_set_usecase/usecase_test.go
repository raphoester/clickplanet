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
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/usecases/save_rule_set_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
)

type playing struct {
	times int
	err   error
}

func (p *playing) Execute(context.Context) (rules.Snapshot, bool, error) {
	p.times++
	return rules.Snapshot{}, false, p.err
}

var rush = rules.Snapshot{Name: "rush", Rules: &antibotv1.Rules{Cohort: &antibotv1.CohortRules{CertainMembers: proto.Uint32(50)}}}

func saving(sets *rules.MemoryStore, plays *playing) *save_rule_set_usecase.UseCase {
	return save_rule_set_usecase.New(cpatomic.FakeTransactioner{}, sets, guard.Parts(), plays)
}

func TestASetIsKeptAsWrittenAndTheSetChosenLastPlaysAgain(t *testing.T) {
	sets := rules.NewMemoryStore(guard.Parts(), rules.Seeded())
	plays := &playing{}
	looser := rules.Named(rules.Default, func(r *antibotv1.Rules) { r.Jury.MinSuspects = proto.Uint32(3) })

	require.NoError(t, saving(sets, plays).Execute(t.Context(), looser))

	kept, ok := sets.Kept(rules.Default)
	require.True(t, ok)
	assert.True(t, proto.Equal(looser.Rules, kept.Rules))
	assert.Equal(t, 1, plays.times, "a save of the set in force takes over at once")
}

func TestASetKeepsOnlyWhatItHolds(t *testing.T) {
	sets := rules.NewMemoryStore(guard.Parts(), rules.Seeded())

	require.NoError(t, saving(sets, &playing{}).Execute(t.Context(), rush))

	kept, ok := sets.Kept("rush")
	require.True(t, ok)
	assert.True(t, proto.Equal(rush.Rules, kept.Rules), "%v", kept.Rules)
}

func TestADefaultThatLeavesAnotherSetUnableToJudgeIsRefusedWithTheSetNamed(t *testing.T) {
	quorum := rules.Snapshot{Name: "quorum", Rules: &antibotv1.Rules{Cohort: &antibotv1.CohortRules{MinMembers: proto.Uint32(5)}}}
	sets := rules.NewMemoryStore(guard.Parts(), rules.Seeded(), quorum)
	plays := &playing{}
	smaller := rules.Named(rules.Default, func(r *antibotv1.Rules) { r.Cohort.CertainMembers = proto.Uint32(4) })

	err := saving(sets, plays).Execute(t.Context(), smaller)

	require.ErrorIs(t, err, rules.ErrInvalidSet)
	assert.Contains(t, err.Error(), `"quorum"`)
	assert.Zero(t, plays.times)
}

func TestASetThatCannotPlayIsNotKept(t *testing.T) {
	sets := rules.NewMemoryStore(guard.Parts(), rules.Seeded())
	plays := &playing{}

	err := saving(sets, plays).Execute(t.Context(),
		rules.Named("rush", func(r *antibotv1.Rules) { r.Cohort.CertainMembers = proto.Uint32(1) }))

	require.ErrorIs(t, err, rules.ErrInvalidSet)
	_, ok := sets.Kept("rush")
	assert.False(t, ok)
	assert.Zero(t, plays.times)
}

func TestASaveThatFailsSaysSo(t *testing.T) {
	sets := rules.NewMemoryStore(guard.Parts())
	down := errors.New("postgres is down")
	sets.FailWith(down)
	plays := &playing{}

	err := saving(sets, plays).Execute(t.Context(), rush)

	require.ErrorIs(t, err, down)
	assert.NotErrorIs(t, err, rules.ErrInvalidSet)
	assert.Zero(t, plays.times)
}

func TestAPlayThatFailsFailsTheSaveAndTheSetStaysKept(t *testing.T) {
	sets := rules.NewMemoryStore(guard.Parts(), rules.Seeded())
	down := errors.New("postgres is down")

	err := saving(sets, &playing{err: down}).Execute(t.Context(), rush)

	require.ErrorIs(t, err, down)
	_, ok := sets.Kept("rush")
	assert.True(t, ok, "the next play or boot plays it")
}
