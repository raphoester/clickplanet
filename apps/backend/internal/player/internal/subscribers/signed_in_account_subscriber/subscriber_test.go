package signed_in_account_subscriber_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/inmemory_player_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/usecases/name_account_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/seasons"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/seasons/inmemory_season_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/subscribers/signed_in_account_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/titles"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/titles/inmemory_title_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/titles/usecases/award_titles_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

func subscriber(store *inmemory_player_store.Store) signed_in_account_subscriber.Subscriber[players.Profile] {
	return signed_in_account_subscriber.New(name_account_usecase.New(
		players.NewGeneratedNames(store, players.NewRepeatedNames("BraveFox42")), store,
		cptime.NewFixedClock(time.Date(2026, 10, 4, 12, 0, 0, 0, time.UTC)),
	))
}

func TestTheAccountSignedInToIsNamedWhetherOrNotTheBrowserHadOne(t *testing.T) {
	for _, previous := range []string{"", "5e0c1b2a-3d4e-4f60-8a71-9b2c3d4e5f60"} {
		store := inmemory_player_store.New()

		err := subscriber(store).Handle(t.Context(),
			&authv1.SignedIn{PreviousAccountId: previous, AccountId: "0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11"})

		require.NoError(t, err)
		account, err := players.AccountIDOf("0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11")
		require.NoError(t, err)
		profile, err := store.Profile(t.Context(), account)
		require.NoError(t, err)
		assert.Equal(t, players.Name("BraveFox42"), profile.TakeSnapshot().Name)
	}
}

func TestAnEventWithNoAccountIsRefused(t *testing.T) {
	err := subscriber(inmemory_player_store.New()).Handle(t.Context(), &authv1.SignedIn{})

	assert.ErrorIs(t, err, players.ErrInvalidAccount)
}

func TestAGuestThatPlayedASeasonGetsItsTitlesWhenItSignsIn(t *testing.T) {
	stats, records, held, accounts := inmemory_player_store.New(), inmemory_season_store.New(), inmemory_title_store.New(), titles.NewFakeAccounts()
	account, err := players.AccountIDOf("0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11")
	require.NoError(t, err)
	now := time.Date(2026, 11, 1, 12, 0, 0, 0, time.UTC)
	accounts.Create(account, players.AccountOf(true, now))
	require.NoError(t, stats.Record(t.Context(), account, players.TakeAt(now.Add(-24*time.Hour))))
	require.NoError(t, records.Save(t.Context(), seasons.NewRecord(account).With(seasons.Season{Number: 0, Flag: "fr", Won: true})))
	award := award_titles_usecase.New(stats, titles.NewCareers(accounts, records), titles.NewBook(held, titles.NewCatalog()), cptime.NewFixedClock(now))

	err = signed_in_account_subscriber.New(award).Handle(t.Context(), &authv1.SignedIn{AccountId: account.String()})

	require.NoError(t, err)
	granted, err := held.Held(t.Context(), account)
	require.NoError(t, err)
	assert.Equal(t, titles.IDs{"champion_s0", "veteran_s0"}, granted, "the map is frozen: no take would grant them")
}
