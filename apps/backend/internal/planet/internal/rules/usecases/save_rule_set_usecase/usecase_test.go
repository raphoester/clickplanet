package save_rule_set_usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules/parts"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules/usecases/save_rule_set_usecase"
)

type playing struct {
	times int
	err   error
}

func (p *playing) Execute(context.Context) (rules.Snapshot, bool, error) {
	p.times++
	return rules.Snapshot{}, false, p.err
}

var rush = rules.Named("rush", func(r *planetv1.Rules) { r.Clicks.Tempo = 3 })

func TestASetIsKeptAsWrittenAndTheSetChosenLastPlaysAgain(t *testing.T) {
	store := rules.NewMemoryStore(parts.All())
	plays := &playing{}

	require.NoError(t, save_rule_set_usecase.New(store, parts.All(), plays).Execute(t.Context(), rush))

	kept, ok := store.Kept("rush")
	require.True(t, ok)
	assert.True(t, proto.Equal(rush.Rules, kept.Rules))
	assert.Equal(t, 1, plays.times, "a save of the set in force takes over at once")
}

func TestASetThatCannotPlayIsNeverKept(t *testing.T) {
	store := rules.NewMemoryStore(parts.All())
	plays := &playing{}
	broken := rules.Named("rush", func(r *planetv1.Rules) { r.Clicks.Tempo = 0 })

	err := save_rule_set_usecase.New(store, parts.All(), plays).Execute(t.Context(), broken)

	require.ErrorIs(t, err, rules.ErrInvalidSet)
	_, kept := store.Kept("rush")
	assert.False(t, kept)
	assert.Zero(t, plays.times)
}

func TestAStoreThatFailsFailsTheSave(t *testing.T) {
	store := rules.NewMemoryStore(parts.All())
	store.FailWith(errors.New("postgres is down"))
	plays := &playing{}

	err := save_rule_set_usecase.New(store, parts.All(), plays).Execute(t.Context(), rush)

	require.Error(t, err)
	assert.Zero(t, plays.times)
}

func TestAPlayThatFailsFailsTheSaveAndTheSetStaysKept(t *testing.T) {
	store := rules.NewMemoryStore(parts.All())
	down := errors.New("postgres is down")

	err := save_rule_set_usecase.New(store, parts.All(), &playing{err: down}).Execute(t.Context(), rush)

	require.ErrorIs(t, err, down)
	_, kept := store.Kept("rush")
	assert.True(t, kept, "the next play or boot plays it")
}
