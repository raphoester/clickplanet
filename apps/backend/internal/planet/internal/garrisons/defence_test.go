package garrisons_test

import (
	"context"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/garrisons"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/garrisons/inmemory_garrison_storage"
)

func storage(t *testing.T) *inmemory_garrison_storage.Storage {
	t.Helper()

	return inmemory_garrison_storage.New(inmemory_garrison_storage.Config{}, 10,
		inmemory_garrison_storage.NewMemoryPersistence(), slog.New(slog.DiscardHandler))
}

func TestEachForeignClickOnADefendedTileTakesOneDefender(t *testing.T) {
	held := storage(t)
	held.Reinforce(7, "pl")
	held.Reinforce(7, "pl")
	defence := garrisons.NewDefence(held)

	assert.Equal(t, clicks.Defended, defence.Strike(7, "pl", "de"))
	assert.Equal(t, clicks.Defended, defence.Strike(7, "pl", "fr"))
	assert.Equal(t, clicks.Taken, defence.Strike(7, "pl", "de"), "with no defender left the next click takes the tile")
}

func TestItsOwnFlagNeverStrikesAGarrison(t *testing.T) {
	held := storage(t)
	held.Reinforce(7, "pl")
	defence := garrisons.NewDefence(held)

	assert.Equal(t, clicks.Unchanged, defence.Strike(7, "pl", "pl"))
	assert.Equal(t, 1, held.Defenders(7, "pl"))
}

func TestTheOutcomeIsForeseenWithoutStriking(t *testing.T) {
	held := storage(t)
	held.Reinforce(7, "pl")
	defence := garrisons.NewDefence(held)

	assert.Equal(t, clicks.Defended, defence.Outcome(7, "pl", "de"))
	assert.Equal(t, clicks.Taken, defence.Outcome(8, "pl", "de"))
	assert.Equal(t, 1, held.Defenders(7, "pl"), "foreseeing a click spends nothing")
}

type owners map[uint32]string

func (o owners) Owner(tile uint32) (string, bool) { return o[tile], true }

type clearer struct{ blasts []clicks.Blast }

func (c *clearer) Clear(_ context.Context, blast clicks.Blast) (clicks.Blast, error) {
	c.blasts = append(c.blasts, blast)
	return blast, nil
}

func TestABombTakesOneDefenderFromEachGarrisonAndClearsTheRest(t *testing.T) {
	held := storage(t)
	held.Reinforce(2, "pl")
	held.Reinforce(2, "pl")
	held.Reinforce(3, "de")
	cleared := &clearer{}
	shelter := garrisons.NewShelter(cleared, owners{1: "pl", 2: "pl", 3: "de", 4: ""}, held)

	blast, err := shelter.Clear(t.Context(), clicks.Blast{Tile: 2, Cleared: []uint32{1, 2, 3, 4}})
	require.NoError(t, err)

	assert.Equal(t, []uint32{1, 4}, blast.Cleared, "a defended tile keeps its flag")
	assert.Equal(t, 1, held.Defenders(2, "pl"))
	assert.Zero(t, held.Defenders(3, "de"))
	require.Len(t, cleared.blasts, 1)
}
