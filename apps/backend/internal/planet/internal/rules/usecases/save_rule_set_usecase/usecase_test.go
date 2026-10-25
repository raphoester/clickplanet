package save_rule_set_usecase_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules/usecases/save_rule_set_usecase"
)

var rush = rules.Snapshot{
	Name:  "rush",
	Tempo: 3,
	Boxes: bonuses.Pace{MinInterval: time.Minute, MaxInterval: 2 * time.Minute, MaxChargesPerHour: 30},
	Crowd: 50,
}

func TestASetIsKeptAsWritten(t *testing.T) {
	store := rules.NewMemoryStore()

	require.NoError(t, save_rule_set_usecase.New(store).Execute(t.Context(), rush))

	kept, ok := store.Kept("rush")
	require.True(t, ok)
	assert.Equal(t, rush, kept)
}

func TestASetThatCannotPlayIsRefusedAndKeepsNothing(t *testing.T) {
	store := rules.NewMemoryStore()
	broken := rush
	broken.Tempo = 0

	err := save_rule_set_usecase.New(store).Execute(t.Context(), broken)

	require.ErrorIs(t, err, rules.ErrInvalidSet)
	_, ok := store.Kept("rush")
	assert.False(t, ok)
}

func TestAStoreThatFailsSaysSo(t *testing.T) {
	store := rules.NewMemoryStore()
	down := errors.New("postgres is down")
	store.FailWith(down)

	require.ErrorIs(t, save_rule_set_usecase.New(store).Execute(t.Context(), rush), down)
}
