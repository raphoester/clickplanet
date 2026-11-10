package warnings_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/warnings"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var at = time.Date(2026, 11, 10, 20, 0, 0, 0, time.UTC)

const (
	ada   = "0b7e5b6c-8f3a-4d2e-9c1a-2f6d8e4b7a10"
	grace = "0b7e5b6c-8f3a-4d2e-9c1a-2f6d8e4b7a11"
)

type enforcing bool

func (e enforcing) Enforcing() bool { return bool(e) }

func bookOf(store *warnings.MemoryStore, teller *warnings.MemoryTeller, enforced enforcing) *warnings.Book {
	return warnings.NewBook(enforced, cptime.NewFixedClock(at), store, cpatomic.FakeTransactioner{}, teller)
}

func TestEachAccountOfARingIsWarnedAndTold(t *testing.T) {
	teller := &warnings.MemoryTeller{}
	book := bookOf(warnings.NewMemoryStore(), teller, true)

	warned, err := book.Warn(t.Context(), []string{ada, grace})

	require.NoError(t, err)
	want := []warnings.Warning{{Account: ada, At: at}, {Account: grace, At: at}}
	assert.Equal(t, want, warned)
	assert.Equal(t, want, teller.Told())
	ringWarned, err := book.Warned(t.Context(), []string{grace})
	require.NoError(t, err)
	assert.True(t, ringWarned)
}

func TestAnAccountIsWarnedOnce(t *testing.T) {
	teller := &warnings.MemoryTeller{}
	store := warnings.NewMemoryStore(warnings.Warning{Account: ada, At: at.Add(-24 * time.Hour)})
	book := bookOf(store, teller, true)

	warned, err := book.Warn(t.Context(), []string{ada, grace})

	require.NoError(t, err)
	assert.Equal(t, []warnings.Warning{{Account: grace, At: at}}, warned)
	assert.Equal(t, []warnings.Warning{{Account: grace, At: at}}, teller.Told())
	warning, ok, err := book.Warning(t.Context(), ada)
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, at.Add(-24*time.Hour), warning.At, "the first warning is kept")
}

func TestAWarningNotEnforcedIsNeitherKeptNorTold(t *testing.T) {
	teller := &warnings.MemoryTeller{}
	book := bookOf(warnings.NewMemoryStore(), teller, false)

	warned, err := book.Warn(t.Context(), []string{ada})

	require.NoError(t, err)
	assert.Empty(t, warned)
	assert.Empty(t, teller.Told())
	ringWarned, err := book.Warned(t.Context(), []string{ada})
	require.NoError(t, err)
	assert.False(t, ringWarned, "kept, it would ban a ring that never saw it")
}

func TestARingWithNoAccountWarnedIsNotWarned(t *testing.T) {
	book := bookOf(warnings.NewMemoryStore(warnings.Warning{Account: ada, At: at}), &warnings.MemoryTeller{}, true)

	warned, err := book.Warned(t.Context(), []string{grace})

	require.NoError(t, err)
	assert.False(t, warned)
}

func TestAWarningThatCannotBeKeptSaysSo(t *testing.T) {
	store := warnings.NewMemoryStore()
	store.FailWith(errors.New("postgres is down"))
	teller := &warnings.MemoryTeller{}

	_, err := bookOf(store, teller, true).Warn(t.Context(), []string{ada})

	require.Error(t, err)
	assert.Empty(t, teller.Told())
}
