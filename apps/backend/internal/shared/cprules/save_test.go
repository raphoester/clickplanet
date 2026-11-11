package cprules_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cprules"
)

type playing struct {
	times int
	err   error
}

func (p *playing) Execute(context.Context) (snapshot, bool, error) {
	p.times++
	return snapshot{}, false, p.err
}

func saving(sets *cprules.MemoryStore[*planetv1.Rules], plays *playing) *cprules.SaveUseCase[*planetv1.Rules] {
	return cprules.NewSaveUseCase(cpatomic.FakeTransactioner{}, sets, parts, plays)
}

func TestASetIsKeptAsWrittenAndTheSetChosenLastPlaysAgain(t *testing.T) {
	sets := keeping(seeded())
	plays := &playing{}

	require.NoError(t, saving(sets, plays).Execute(t.Context(), rush()))

	kept, ok := sets.Kept("rush")
	require.True(t, ok)
	assert.True(t, proto.Equal(rush().Rules, kept.Rules), "a set keeps only what it holds")
	assert.Equal(t, 1, plays.times, "a save of the set in force takes over at once")
}

func TestASetThatCannotPlayIsNeverKept(t *testing.T) {
	sets := keeping(seeded())
	plays := &playing{}
	broken := snapshot{Name: "rush", Rules: &planetv1.Rules{Clicks: &planetv1.ClickRules{Tempo: proto.Float64(0)}}}

	err := saving(sets, plays).Execute(t.Context(), broken)

	require.ErrorIs(t, err, cprules.ErrInvalidSet)
	assert.EqualError(t, err, `cannot keep the rule set "rush": this rule set cannot play: clicks.tempo is 0, and must be above 0`)
	_, kept := sets.Kept("rush")
	assert.False(t, kept)
	assert.Zero(t, plays.times)
}

func TestADefaultIsKeptWhenEveryOtherSetStillPlaysOverIt(t *testing.T) {
	sets := keeping(seeded(), rush())
	slower := named(cprules.Default, func(r *planetv1.Rules) { r.Clicks.Burst = proto.Uint32(40) })

	require.NoError(t, saving(sets, &playing{}).Execute(t.Context(), slower))

	kept, _ := sets.Kept(cprules.Default)
	assert.True(t, proto.Equal(slower.Rules, kept.Rules))
}

func TestADefaultThatLeavesAnotherSetUnableToPlayIsRefusedWithTheSetNamed(t *testing.T) {
	quick := snapshot{Name: "quick", Rules: &planetv1.Rules{Clicks: &planetv1.ClickRules{NewAccountClicks: proto.Float64(50)}}}
	sets := keeping(seeded(), quick, rush())
	plays := &playing{}
	slower := named(cprules.Default, func(r *planetv1.Rules) { r.Clicks.Burst = proto.Uint32(40) })

	err := saving(sets, plays).Execute(t.Context(), slower)

	require.ErrorIs(t, err, cprules.ErrInvalidSet)
	assert.Contains(t, err.Error(), `"quick"`)
	assert.Contains(t, err.Error(), "clicks.newAccountClicks is 50, over burst (40)")
	assert.NotContains(t, err.Error(), `"rush"`)
	assert.Zero(t, plays.times)
}

func TestASetIsRefusedWithoutADefaultToPlayOver(t *testing.T) {
	err := saving(keeping(), &playing{}).Execute(t.Context(), rush())

	require.ErrorIs(t, err, cprules.ErrUnknownSet)
}

func TestAStoreThatFailsFailsTheSave(t *testing.T) {
	sets := keeping(seeded())
	down := errors.New("postgres is down")
	sets.FailWith(down)
	plays := &playing{}

	err := saving(sets, plays).Execute(t.Context(), rush())

	require.ErrorIs(t, err, down)
	assert.NotErrorIs(t, err, cprules.ErrInvalidSet)
	assert.Zero(t, plays.times)
}

func TestAPlayThatFailsFailsTheSaveAndTheSetStaysKept(t *testing.T) {
	sets := keeping(seeded())
	down := errors.New("postgres is down")

	err := saving(sets, &playing{err: down}).Execute(t.Context(), rush())

	require.ErrorIs(t, err, down)
	_, kept := sets.Kept("rush")
	assert.True(t, kept, "the next play or boot plays it")
}
