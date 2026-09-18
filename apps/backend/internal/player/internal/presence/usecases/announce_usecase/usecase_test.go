package announce_usecase_test

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
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/presence/usecases/announce_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcountries"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var (
	now   = time.Date(2026, 9, 17, 12, 0, 0, 0, time.UTC)
	ada   = players.AccountID{15: 1}
	guest = players.AccountID{15: 2}
)

type fixture struct {
	store   *inmemory_player_store.Store
	visits  *inmemory_visit_storage.Storage
	useCase *announce_usecase.UseCase
}

func setup(t *testing.T) fixture {
	t.Helper()

	clock := cptime.NewFixedClock(now)
	store := inmemory_player_store.New()
	require.NoError(t, store.SaveProfile(t.Context(), players.Profile{Account: ada, Name: "Ada_L", UpdatedAt: now}))
	visits := inmemory_visit_storage.New(clock)
	authors := get_author_usecase.New(store, players.NewGuestCodes(store, &players.SequentialCodes{}))

	return fixture{
		store:   store,
		visits:  visits,
		useCase: announce_usecase.New(authors, visits, cpcountries.New(), clock, "pepper"),
	}
}

func TestAnAccountWithAUsernameIsRecordedUnderIt(t *testing.T) {
	f := setup(t)

	err := f.useCase.Execute(t.Context(), announce_usecase.In{Account: ada, Country: "fr", IP: "1.2.3.4"})

	require.NoError(t, err)
	assert.Equal(t, []presence.Visit{{
		Account: ada,
		Key:     "1",
		Author:  players.Author{Name: "Ada_L"},
		Tag:     players.TagOf("pepper", "1.2.3.4"),
		Country: "fr",
		At:      now,
	}}, f.visits.Visits())
}

func TestAnAdminIsRecordedAsOne(t *testing.T) {
	f := setup(t)
	f.store.MakeAdmin(ada)

	err := f.useCase.Execute(t.Context(), announce_usecase.In{Account: ada, Country: "fr", IP: "1.2.3.4"})

	require.NoError(t, err)
	require.Len(t, f.visits.Visits(), 1)
	assert.True(t, f.visits.Visits()[0].Author.Admin)
}

func TestAGuestIsRecordedUnderItsCodeWhateverItsAddress(t *testing.T) {
	f := setup(t)

	require.NoError(t, f.useCase.Execute(t.Context(), announce_usecase.In{Account: guest, Country: "de", IP: "1.2.3.4"}))
	require.NoError(t, f.useCase.Execute(t.Context(), announce_usecase.In{Account: guest, Country: "de", IP: "5.6.7.8"}))

	require.Len(t, f.visits.Visits(), 1)
	assert.Equal(t, players.Author{Name: "guest_000001", Guest: true}, f.visits.Visits()[0].Author,
		"a new network is not a new name")
}

func TestAnUnknownCountryIsRefusedAndNotRecorded(t *testing.T) {
	f := setup(t)

	err := f.useCase.Execute(t.Context(), announce_usecase.In{Account: ada, Country: "atlantis", IP: "1.2.3.4"})

	require.ErrorIs(t, err, presence.ErrUnknownCountry)
	assert.Empty(t, f.visits.Visits())
}

func TestAStoreFailureIsAnErrorAndNotRecorded(t *testing.T) {
	f := setup(t)
	f.store.FailWith(errors.New("postgres is down"))

	err := f.useCase.Execute(t.Context(), announce_usecase.In{Account: ada, Country: "fr", IP: "1.2.3.4"})

	require.Error(t, err)
	assert.NotErrorIs(t, err, presence.ErrUnknownCountry)
	assert.Empty(t, f.visits.Visits())
}
