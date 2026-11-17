package eras_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/eras"
)

var earth2 = eras.Board{ID: eras.Earth1.ID + 1, Name: "earth-2", Hash: "0123abcd"}

func TestTheBootPlaysTheWorldOnTheBoardItsEraNames(t *testing.T) {
	store := eras.NewMemoryStore()
	boards, err := eras.BoardsOf([]eras.Board{eras.Earth1}, eras.Earth1Carried)
	require.NoError(t, err)

	stage, err := eras.Playing(t.Context(), store, boards, clicks.World)
	require.NoError(t, err)
	assert.Equal(t, clicks.World.On(eras.Earth1.ID), stage)
	assert.Equal(t, clicks.BoardName("earth-1"), boards.Name(stage.Board))
}

func TestTheBootRefusesAMapWhoseEraPlaysABoardThisProcessDoesNotCarry(t *testing.T) {
	store := eras.NewMemoryStore()
	store.Keep(earth2)
	require.NoError(t, store.Start(t.Context(), clicks.World.On(earth2.ID), time.Date(2026, 11, 24, 3, 0, 0, 0, time.UTC)))
	boards, err := eras.BoardsOf([]eras.Board{eras.Earth1, earth2}, eras.Earth1Carried)
	require.NoError(t, err)

	_, err = eras.Playing(t.Context(), store, boards, clicks.World)

	require.ErrorIs(t, err, eras.ErrNotCarried)
	assert.Contains(t, err.Error(), "earth-2")
}

func TestTheBootRefusesAMapWithNoEra(t *testing.T) {
	boards, err := eras.BoardsOf([]eras.Board{eras.Earth1}, eras.Earth1Carried)
	require.NoError(t, err)

	_, err = eras.Playing(t.Context(), eras.NewMemoryStore(), boards, "battle")

	require.ErrorIs(t, err, eras.ErrNoEra)
}

func TestABoardCarriedIsOneWithARowOfTheSameHash(t *testing.T) {
	_, err := eras.BoardsOf([]eras.Board{eras.Earth1}, eras.Carried{Name: "earth-1", Hash: "deadbeef"})
	require.ErrorIs(t, err, eras.ErrUnkept, "a board regenerated under its name, its tiles moved, is another board")

	_, err = eras.BoardsOf([]eras.Board{eras.Earth1}, eras.Carried{Name: "earth-2", Hash: earth2.Hash})
	require.ErrorIs(t, err, eras.ErrUnkept, "a board shipped with no migration")

	boards, err := eras.BoardsOf([]eras.Board{eras.Earth1, earth2}, eras.Earth1Carried)
	require.NoError(t, err)
	_, carried := boards.Board(earth2.ID)
	assert.False(t, carried, "a board planet keeps is not one this process carries")
	assert.Empty(t, boards.Name(earth2.ID))
}
