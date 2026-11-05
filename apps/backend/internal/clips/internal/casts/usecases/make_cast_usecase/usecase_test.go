package make_cast_usecase_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/casts"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/casts/usecases/make_cast_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var now = time.Date(2026, 10, 28, 12, 0, 0, 0, time.UTC)

func account(t *testing.T, last string) casts.AccountID {
	t.Helper()
	id, err := casts.AccountIDOf("0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a" + last)
	require.NoError(t, err)
	return id
}

func TestTheCastIsKeptAndItsTakesAnswerUnderNumbers(t *testing.T) {
	ada, bob := account(t, "01"), account(t, "02")
	ledger := casts.NewScriptedLedger(
		casts.Take{At: now.Add(-time.Hour), Tile: 7, Country: "fr", Previous: "de", Account: ada},
		casts.Take{At: now.Add(-time.Hour), Tile: 8, Country: "de", Previous: "fr", Account: bob},
		casts.Take{At: now.Add(-time.Hour), Tile: 9, Country: "fr", Previous: "it", Account: ada},
	)
	names := casts.NewScriptedNames(map[casts.AccountID]string{ada: "Ada"})
	store := casts.NewMemoryStore()

	cast, takes, err := make_cast_usecase.New(ledger, names, store, &casts.SequentialIDs{}, cptime.NewFixedClock(now)).
		Execute(t.Context(), now.Add(-12*time.Hour), time.Time{})
	require.NoError(t, err)

	assert.Equal(t, []casts.Window{{Since: now.Add(-12 * time.Hour), Until: now}}, ledger.Asked)
	assert.Equal(t, [][]casts.AccountID{{ada, bob}}, names.Asked, "each account is asked once")
	assert.Equal(t, []casts.NumberedTake{
		{At: now.Add(-time.Hour), Tile: 7, Country: "fr", Previous: "de", Fighter: 1},
		{At: now.Add(-time.Hour), Tile: 9, Country: "fr", Previous: "it", Fighter: 1},
	}, takes, "bob keeps his name out of the clips")
	kept, err := store.Cast(t.Context(), cast.TakeSnapshot().ID)
	require.NoError(t, err)
	assert.Equal(t, cast.TakeSnapshot(), kept.TakeSnapshot())
}

func TestTheCastsTooOldToBePostedAreForgotten(t *testing.T) {
	store := casts.NewMemoryStore()
	old := casts.CastOf(casts.CastID{9}, now.Add(-73*time.Hour), casts.Window{}, nil, nil)
	recent := casts.CastOf(casts.CastID{8}, now.Add(-time.Hour), casts.Window{}, nil, nil)
	require.NoError(t, store.Keep(t.Context(), old))
	require.NoError(t, store.Keep(t.Context(), recent))

	_, _, err := make_cast_usecase.New(casts.NewScriptedLedger(), casts.NewScriptedNames(nil), store, &casts.SequentialIDs{},
		cptime.NewFixedClock(now)).Execute(t.Context(), now.Add(-time.Hour), now)
	require.NoError(t, err)

	_, err = store.Cast(t.Context(), casts.CastID{9})
	require.ErrorIs(t, err, casts.ErrNoCast)
	_, err = store.Cast(t.Context(), casts.CastID{8})
	require.NoError(t, err)
}

func TestAWindowTheLedgerCannotReadAsksNothing(t *testing.T) {
	ledger := casts.NewScriptedLedger()

	_, _, err := make_cast_usecase.New(ledger, casts.NewScriptedNames(nil), casts.NewMemoryStore(), &casts.SequentialIDs{},
		cptime.NewFixedClock(now)).Execute(t.Context(), now.Add(-80*time.Hour), now)

	require.ErrorIs(t, err, casts.ErrInvalidWindow)
	assert.Empty(t, ledger.Asked)
}

func TestALedgerThatCannotBeReadKeepsNoCast(t *testing.T) {
	ledger := casts.NewScriptedLedger()
	ledger.FailWith(errors.New("planet is down"))
	store := casts.NewMemoryStore()

	_, _, err := make_cast_usecase.New(ledger, casts.NewScriptedNames(nil), store, &casts.SequentialIDs{},
		cptime.NewFixedClock(now)).Execute(t.Context(), now.Add(-time.Hour), now)

	require.ErrorContains(t, err, "planet is down")
	ids, err := store.MadeBefore(t.Context(), now.Add(time.Hour))
	require.NoError(t, err)
	assert.Empty(t, ids)
}
