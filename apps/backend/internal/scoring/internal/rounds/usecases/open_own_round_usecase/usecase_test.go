package open_own_round_usecase_test

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/calendar"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/rounds"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/rounds/inmemory_round_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/rounds/usecases/open_own_round_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

func utc(hour int) time.Time {
	return time.Date(2026, 10, 31, hour, 0, 0, 0, time.UTC)
}

var seasonZero = calendar.New(calendar.Config{DailyCutoff: 21 * time.Hour}, calendar.Window{Number: 0, StartsAt: time.Date(2026, 10, 3, 0, 0, 0, 0, time.UTC), EndsAt: utc(23)})

var battle = rounds.EntryID(uuid.MustParse("0192f5a0-7c00-7000-8000-000000000031"))

type waker struct{ woken int }

func (w *waker) Wake() { w.woken++ }

func TestAWindowOpensItsRoundAndWakesTheRunner(t *testing.T) {
	store, woken := inmemory_round_store.New(), &waker{}

	err := open_own_round_usecase.New(seasonZero, store, cptime.NewFixedClock(utc(21)), woken).
		Execute(t.Context(), battle, 3, utc(21), utc(23))

	require.NoError(t, err)
	unclosed, err := store.Unclosed(t.Context(), utc(23))
	require.NoError(t, err)
	assert.Equal(t, []rounds.Round{{EndsAt: utc(23), Finale: true, Own: true, Entry: battle, StartsAt: utc(21), Times: 3}}, unclosed)
	assert.Equal(t, 1, woken.woken)
}

func TestAWindowHeardOfOnceItEndedOpensNothing(t *testing.T) {
	store, woken := inmemory_round_store.New(), &waker{}

	err := open_own_round_usecase.New(seasonZero, store, cptime.NewFixedClock(utc(23)), woken).
		Execute(t.Context(), battle, 3, utc(21), utc(23))

	require.NoError(t, err)
	first, found, err := store.FirstEnd(t.Context())
	require.NoError(t, err)
	assert.False(t, found, "%s", first)
	assert.Zero(t, woken.woken)
}

func TestARoundThatCannotBeOpenedIsAnError(t *testing.T) {
	store := inmemory_round_store.New()
	store.FailWith(errors.New("postgres is down"))

	err := open_own_round_usecase.New(seasonZero, store, cptime.NewFixedClock(utc(21)), &waker{}).
		Execute(t.Context(), battle, 3, utc(21), utc(23))

	require.Error(t, err)
}
