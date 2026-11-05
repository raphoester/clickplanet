package save_rule_set_usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules/parts"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules/usecases/save_rule_set_usecase"
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

var rush = rules.Snapshot{Name: "rush", Rules: &planetv1.Rules{Clicks: &planetv1.ClickRules{Tempo: proto.Float64(3)}}}

func saving(store *rules.MemoryStore, plays *playing) *save_rule_set_usecase.UseCase {
	return save_rule_set_usecase.New(cpatomic.FakeTransactioner{}, store, parts.All(), plays)
}

func TestASetIsKeptAsWrittenAndTheSetChosenLastPlaysAgain(t *testing.T) {
	store := rules.NewMemoryStore(parts.All(), rules.Seeded())
	plays := &playing{}

	require.NoError(t, saving(store, plays).Execute(t.Context(), rush))

	kept, ok := store.Kept("rush")
	require.True(t, ok)
	assert.True(t, proto.Equal(rush.Rules, kept.Rules), "a set keeps only what it holds")
	assert.Equal(t, 1, plays.times, "a save of the set in force takes over at once")
}

func TestASetThatCannotPlayIsNeverKept(t *testing.T) {
	store := rules.NewMemoryStore(parts.All(), rules.Seeded())
	plays := &playing{}
	broken := rules.Snapshot{Name: "rush", Rules: &planetv1.Rules{Clicks: &planetv1.ClickRules{Tempo: proto.Float64(0)}}}

	err := saving(store, plays).Execute(t.Context(), broken)

	require.ErrorIs(t, err, rules.ErrInvalidSet)
	_, kept := store.Kept("rush")
	assert.False(t, kept)
	assert.Zero(t, plays.times)
}

func TestADefaultIsKeptWhenEveryOtherSetStillPlaysOverIt(t *testing.T) {
	store := rules.NewMemoryStore(parts.All(), rules.Seeded(), rush)
	slower := rules.Named(rules.Default, func(r *planetv1.Rules) { r.Clicks.Burst = proto.Uint32(40) })

	require.NoError(t, saving(store, &playing{}).Execute(t.Context(), slower))

	kept, _ := store.Kept(rules.Default)
	assert.True(t, proto.Equal(slower.Rules, kept.Rules))
}

func TestADefaultThatLeavesAnotherSetUnableToPlayIsRefusedWithTheSetNamed(t *testing.T) {
	quick := rules.Snapshot{Name: "quick", Rules: &planetv1.Rules{
		Bonuses: &planetv1.BonusRules{Boxes: &planetv1.BoxPace{MaxInterval: durationpb.New(5 * time.Minute)}},
	}}
	store := rules.NewMemoryStore(parts.All(), rules.Seeded(), quick)
	plays := &playing{}
	slower := rules.Named(rules.Default, func(r *planetv1.Rules) { r.Bonuses.Boxes.MinInterval = durationpb.New(6 * time.Minute) })

	err := saving(store, plays).Execute(t.Context(), slower)

	require.ErrorIs(t, err, rules.ErrInvalidSet)
	assert.Contains(t, err.Error(), `"quick"`)
	assert.Zero(t, plays.times)
}

func TestASetIsRefusedWithoutADefaultToPlayOver(t *testing.T) {
	store := rules.NewMemoryStore(parts.All())

	err := saving(store, &playing{}).Execute(t.Context(), rush)

	require.ErrorIs(t, err, rules.ErrUnknownSet)
}

func TestAStoreThatFailsFailsTheSave(t *testing.T) {
	store := rules.NewMemoryStore(parts.All(), rules.Seeded())
	store.FailWith(errors.New("postgres is down"))
	plays := &playing{}

	err := saving(store, plays).Execute(t.Context(), rush)

	require.Error(t, err)
	assert.Zero(t, plays.times)
}

func TestAPlayThatFailsFailsTheSaveAndTheSetStaysKept(t *testing.T) {
	store := rules.NewMemoryStore(parts.All(), rules.Seeded())
	down := errors.New("postgres is down")

	err := saving(store, &playing{err: down}).Execute(t.Context(), rush)

	require.ErrorIs(t, err, down)
	_, kept := store.Kept("rush")
	assert.True(t, kept, "the next play or boot plays it")
}
