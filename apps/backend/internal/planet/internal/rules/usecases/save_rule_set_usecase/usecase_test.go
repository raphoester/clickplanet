package save_rule_set_usecase_test

import (
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

type playing rules.SetName

func (p playing) Plays(name rules.SetName) bool { return name == rules.SetName(p) }

var rush = rules.Named("rush", func(r *planetv1.Rules) { r.Clicks.Tempo = 3 })

func TestASetIsKeptAsWritten(t *testing.T) {
	store := rules.NewMemoryStore(parts.All())

	require.NoError(t, save_rule_set_usecase.New(store, parts.All(), playing(rules.Default)).Execute(t.Context(), rush))

	kept, ok := store.Kept("rush")
	require.True(t, ok)
	assert.True(t, proto.Equal(rush.Rules, kept.Rules))
}

func TestASetThatCannotPlayIsNeverKept(t *testing.T) {
	store := rules.NewMemoryStore(parts.All())
	broken := rules.Named("rush", func(r *planetv1.Rules) { r.Clicks.Tempo = 0 })

	err := save_rule_set_usecase.New(store, parts.All(), playing(rules.Default)).Execute(t.Context(), broken)

	require.ErrorIs(t, err, rules.ErrInvalidSet)
	_, kept := store.Kept("rush")
	assert.False(t, kept)
}

func TestTheSetThatPlaysNowIsNeverKept(t *testing.T) {
	store := rules.NewMemoryStore(parts.All())

	err := save_rule_set_usecase.New(store, parts.All(), playing(rules.Default)).
		Execute(t.Context(), rules.Named(rules.Default, func(r *planetv1.Rules) { r.Clicks.Tempo = 3 }))

	require.ErrorIs(t, err, save_rule_set_usecase.ErrPlaying)
	_, kept := store.Kept(rules.Default)
	assert.False(t, kept)
}

func TestAStoreThatFailsFailsTheSave(t *testing.T) {
	store := rules.NewMemoryStore(parts.All())
	store.FailWith(errors.New("postgres is down"))

	err := save_rule_set_usecase.New(store, parts.All(), playing(rules.Default)).Execute(t.Context(), rush)

	require.Error(t, err)
}
