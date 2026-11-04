package move_visit_usecase_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/inmemory_player_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/usecases/get_author_usecase"
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

func keyless(visits []presence.Visit) []presence.Visit {
	for i := range visits {
		visits[i] = visits[i].Keyed("")
	}
	return visits
}

func guestVisit() presence.Visit {
	return presence.NewVisit(guest, players.GuestAuthor("0b1c2d"), "aaaaaa", "fr", now)
}

func useCase(store *inmemory_player_store.Store, visits *inmemory_visit_storage.Storage) *move_visit_usecase.UseCase {
	authors := get_author_usecase.New(store, players.NewGuestCodes(store, &players.SequentialCodes{}))
	return move_visit_usecase.New(authors, visits)
}

func TestSigningInToAKnownAccountShowsItsUsernameInPlaceOfTheGuest(t *testing.T) {
	store := inmemory_player_store.New()
	require.NoError(t, store.SaveProfile(t.Context(), players.NewProfile(ada, "Ada_L", now)))
	visits := inmemory_visit_storage.New(cptime.NewFixedClock(now))
	visits.Record(guestVisit())

	err := useCase(store, visits).Execute(t.Context(), guest, ada)

	require.NoError(t, err)
	assert.Equal(t, []presence.Visit{guestVisit().For(ada, players.NamedAuthor(players.ProfileSnapshot{Name: "Ada_L"}.Restore()))}, keyless(visits.Visits()))
}

func TestSigningInToANewAccountShowsItsOwnGuestCode(t *testing.T) {
	store := inmemory_player_store.New()
	visits := inmemory_visit_storage.New(cptime.NewFixedClock(now))
	visits.Record(guestVisit())

	err := useCase(store, visits).Execute(t.Context(), guest, ada)

	require.NoError(t, err)
	assert.Equal(t, []presence.Visit{guestVisit().For(ada, players.GuestAuthor("000001"))},
		keyless(visits.Visits()), "the code of the account the browser is on now, not the one it left")
}

func TestASignInThatKeepsTheAccountChangesNothingAndReadsNothing(t *testing.T) {
	store := inmemory_player_store.New()
	store.FailWith(errors.New("postgres is down"))
	visits := inmemory_visit_storage.New(cptime.NewFixedClock(now))
	visits.Record(guestVisit())

	err := useCase(store, visits).Execute(t.Context(), guest, guest)

	require.NoError(t, err)
	assert.Equal(t, []presence.Visit{guestVisit()}, keyless(visits.Visits()), "no username yet: SetName shows it")
}

func TestAStoreFailureIsAnErrorAndMovesNothing(t *testing.T) {
	store := inmemory_player_store.New()
	store.FailWith(errors.New("postgres is down"))
	visits := inmemory_visit_storage.New(cptime.NewFixedClock(now))
	visits.Record(guestVisit())

	err := useCase(store, visits).Execute(t.Context(), guest, ada)

	require.Error(t, err)
	assert.Equal(t, []presence.Visit{guestVisit()}, keyless(visits.Visits()))
}

func TestSigningInToAnAdminShowsItsCrown(t *testing.T) {
	store := inmemory_player_store.New()
	require.NoError(t, store.SaveProfile(t.Context(), players.NewProfile(ada, "Ada_L", now)))
	store.MakeAdmin(ada)
	visits := inmemory_visit_storage.New(cptime.NewFixedClock(now))
	visits.Record(guestVisit())

	err := useCase(store, visits).Execute(t.Context(), guest, ada)

	require.NoError(t, err)
	require.Len(t, visits.Visits(), 1)
	assert.True(t, visits.Visits()[0].Author().Admin())
}
