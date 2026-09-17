package move_visit_usecase_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/inmemory_player_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/presence"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/presence/inmemory_visit_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/presence/usecases/move_visit_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var (
	now   = time.Date(2026, 9, 17, 12, 0, 0, 0, time.UTC)
	ada   = players.AccountID{15: 1}
	guest = players.AccountID{15: 2}
)

// keyless is the visits as announced, before the storage keyed them.
func keyless(visits []presence.Visit) []presence.Visit {
	for i := range visits {
		visits[i].Key = ""
	}
	return visits
}

func guestVisit() presence.Visit {
	return presence.Visit{Account: guest, GuestName: "Bob", Tag: "aaaaaa", Country: "fr", At: now}
}

func TestSigningInToAKnownAccountShowsItsUsernameInPlaceOfTheGuest(t *testing.T) {
	store := inmemory_player_store.New()
	require.NoError(t, store.SaveProfile(t.Context(), players.Profile{Account: ada, Name: "Ada_L", UpdatedAt: now}))
	visits := inmemory_visit_storage.New(cptime.NewFixedClock(now))
	visits.Record(guestVisit())

	err := move_visit_usecase.New(store, visits).Execute(t.Context(), guest, ada)

	require.NoError(t, err)
	assert.Equal(t, []presence.Visit{guestVisit().For(ada, "Ada_L")}, keyless(visits.Visits()))
}

func TestASignInThatKeepsTheAccountChangesNothingAndReadsNothing(t *testing.T) {
	store := inmemory_player_store.New()
	store.FailWith(errors.New("postgres is down"))
	visits := inmemory_visit_storage.New(cptime.NewFixedClock(now))
	visits.Record(guestVisit())

	err := move_visit_usecase.New(store, visits).Execute(t.Context(), guest, guest)

	require.NoError(t, err)
	assert.Equal(t, []presence.Visit{guestVisit()}, keyless(visits.Visits()), "no username yet: SetName shows it")
}

func TestAStoreFailureIsAnErrorAndMovesNothing(t *testing.T) {
	store := inmemory_player_store.New()
	store.FailWith(errors.New("postgres is down"))
	visits := inmemory_visit_storage.New(cptime.NewFixedClock(now))
	visits.Record(guestVisit())

	err := move_visit_usecase.New(store, visits).Execute(t.Context(), guest, ada)

	require.Error(t, err)
	assert.Equal(t, []presence.Visit{guestVisit()}, keyless(visits.Visits()))
}

func TestSigningInToAnAdminShowsItsCrown(t *testing.T) {
	store := inmemory_player_store.New()
	require.NoError(t, store.SaveProfile(t.Context(), players.Profile{Account: ada, Name: "Ada_L", UpdatedAt: now}))
	store.MakeAdmin(ada)
	visits := inmemory_visit_storage.New(cptime.NewFixedClock(now))
	visits.Record(guestVisit())

	err := move_visit_usecase.New(store, visits).Execute(t.Context(), guest, ada)

	require.NoError(t, err)
	require.Len(t, visits.Visits(), 1)
	assert.True(t, visits.Visits()[0].Admin)
}
