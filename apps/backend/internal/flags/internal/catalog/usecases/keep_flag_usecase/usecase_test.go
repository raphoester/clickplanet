package keep_flag_usecase_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/catalog"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/catalog/usecases/keep_flag_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
)

var openedAt = time.Date(2026, 11, 23, 9, 0, 0, 0, time.UTC)

func TestAFlagAPlayerPlayedForIsKeptForGood(t *testing.T) {
	store := catalog.NewMemoryStore()
	require.NoError(t, store.Save(t.Context(), catalog.OpenFlag("r-0000000001", "Rennes", openedAt)))
	useCase := keep_flag_usecase.New(cpatomic.FakeTransactioner{}, store)

	require.NoError(t, useCase.Execute(t.Context(), keep_flag_usecase.In{Flag: "r-0000000001"}))
	require.NoError(t, useCase.Execute(t.Context(), keep_flag_usecase.In{Flag: "r-0000000001"}), "told twice")

	kept, err := store.Flag(t.Context(), "r-0000000001")
	require.NoError(t, err)
	assert.True(t, kept.TakeSnapshot().Played)
}

func TestAFlagDeletedOrNeverOpenedIsNoError(t *testing.T) {
	useCase := keep_flag_usecase.New(cpatomic.FakeTransactioner{}, catalog.NewMemoryStore())

	require.NoError(t, useCase.Execute(t.Context(), keep_flag_usecase.In{Flag: "r-0000000001"}))
	require.ErrorIs(t, useCase.Execute(t.Context(), keep_flag_usecase.In{Flag: "fr"}), catalog.ErrInvalidFlagID)
}
