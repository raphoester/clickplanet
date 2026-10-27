package play_season_usecase_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/calendar"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/calendar/inmemory_season_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/calendar/usecases/play_season_usecase"
)

var (
	starts  = time.Date(2026, 10, 3, 0, 0, 0, 0, time.UTC)
	ends    = time.Date(2026, 10, 31, 23, 0, 0, 0, time.UTC)
	decided = time.Date(2026, 10, 28, 12, 0, 0, 0, time.UTC)
	zero    = calendar.Decision{Window: calendar.Window{Number: 0, StartsAt: starts, EndsAt: ends}, DecidedAt: decided}
)

type waker struct{ woken int }

func (w *waker) Wake() { w.woken++ }

type harness struct {
	store    *inmemory_season_store.Store
	calendar calendar.Calendar
	waker    *waker
	useCase  *play_season_usecase.UseCase
}

func newHarness() harness {
	h := harness{store: inmemory_season_store.New(), calendar: calendar.New(calendar.Config{DailyCutoff: 21 * time.Hour}), waker: &waker{}}
	h.useCase = play_season_usecase.New(h.store, h.calendar, h.waker)
	return h
}

func TestASeasonToldIsKeptAndRunsAndTheRunnerWakes(t *testing.T) {
	h := newHarness()

	took, err := h.useCase.Execute(t.Context(), zero)

	require.NoError(t, err)
	assert.True(t, took)
	season, ok := h.calendar.Current(decided)
	require.True(t, ok)
	assert.Equal(t, ends, season.EndsAt)
	kept, err := h.store.Decisions(t.Context())
	require.NoError(t, err)
	assert.Equal(t, []calendar.Decision{zero}, kept)
	assert.Equal(t, 1, h.waker.woken)
}

func TestTheLatestDecisionWins(t *testing.T) {
	h := newHarness()
	moved := zero
	moved.Window.EndsAt, moved.DecidedAt = ends.AddDate(0, 0, 7), decided.Add(time.Hour)
	_, err := h.useCase.Execute(t.Context(), moved)
	require.NoError(t, err)

	took, err := h.useCase.Execute(t.Context(), zero)

	require.NoError(t, err)
	assert.False(t, took, "an older decision arrives late")
	season, ok := h.calendar.Current(ends)
	require.True(t, ok)
	assert.Equal(t, moved.Window.EndsAt, season.EndsAt)
}

func TestADecisionSentAgainChangesNothing(t *testing.T) {
	h := newHarness()
	_, err := h.useCase.Execute(t.Context(), zero)
	require.NoError(t, err)

	took, err := h.useCase.Execute(t.Context(), zero)

	require.NoError(t, err)
	assert.False(t, took)
	assert.Equal(t, 1, h.waker.woken)
}

func TestAWindowThatEndsBeforeItStartsIsRefused(t *testing.T) {
	h := newHarness()
	wrong := zero
	wrong.Window.EndsAt = starts

	_, err := h.useCase.Execute(t.Context(), wrong)

	require.ErrorIs(t, err, calendar.ErrInvalidSeason)
	kept, err := h.store.Decisions(t.Context())
	require.NoError(t, err)
	assert.Empty(t, kept)
}

func TestSeasonsThatCannotBeReadPlayNothing(t *testing.T) {
	h := newHarness()
	down := errors.New("postgres is down")
	h.store.FailWith(down)

	_, err := h.useCase.Execute(t.Context(), zero)

	require.ErrorIs(t, err, down)
	_, ok := h.calendar.Current(decided)
	assert.False(t, ok)
}
