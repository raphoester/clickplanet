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
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/subscribers/signed_in_account_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

func subscriber(store *inmemory_player_store.Store) signed_in_account_subscriber.Subscriber {
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
		assert.Equal(t, players.Name("BraveFox42"), profile.Name)
	}
}

func TestAnEventWithNoAccountIsRefused(t *testing.T) {
	err := subscriber(inmemory_player_store.New()).Handle(t.Context(), &authv1.SignedIn{})

	assert.ErrorIs(t, err, players.ErrInvalidAccount)
}
