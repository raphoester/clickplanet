package save_rule_set_usecase_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/rules"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/rules/usecases/save_rule_set_usecase"
)

type playing rules.SetName

func (p playing) Plays(name rules.SetName) bool { return name == rules.SetName(p) }

func TestASetIsKeptAsWritten(t *testing.T) {
	store := rules.NewMemoryStore()

	require.NoError(t, save_rule_set_usecase.New(store, playing(rules.Default)).Execute(t.Context(), rules.Battle("rush", 3)))

	kept, ok := store.Kept("rush")
	require.True(t, ok)
	assert.True(t, proto.Equal(rules.Battle("rush", 3).Rules, kept.Rules))
}

func TestASetThatCannotPlayIsNeverKept(t *testing.T) {
	store := rules.NewMemoryStore()

	err := save_rule_set_usecase.New(store, playing(rules.Default)).Execute(t.Context(), rules.Battle("rush", 0))

	require.ErrorIs(t, err, rules.ErrInvalidSet)
	_, kept := store.Kept("rush")
	assert.False(t, kept)
}

func TestTheSetThatPlaysNowIsNeverKept(t *testing.T) {
	store := rules.NewMemoryStore()

	err := save_rule_set_usecase.New(store, playing("rush")).Execute(t.Context(), rules.Battle("rush", 3))

	require.ErrorIs(t, err, save_rule_set_usecase.ErrPlaying)
	_, kept := store.Kept("rush")
	assert.False(t, kept)
}

func TestAStoreThatFailsFailsTheSave(t *testing.T) {
	store := rules.NewMemoryStore()
	store.FailWith(errors.New("postgres is down"))

	err := save_rule_set_usecase.New(store, playing(rules.Default)).Execute(t.Context(), rules.Battle("rush", 3))

	require.Error(t, err)
}
