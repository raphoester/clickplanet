package keep_past_name_usecase_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/pastnames"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/pastnames/inmemory_past_name_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/pastnames/usecases/keep_past_name_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/inmemory_player_store"
)

var (
	ada    = players.AccountID{15: 1}
	monday = time.Date(2026, 11, 16, 20, 0, 0, 0, time.UTC)
)

func leftByAda(t *testing.T) pastnames.PastName {
	t.Helper()

	name, err := pastnames.Snapshot{Account: ada, Name: "BraveFox42", LeftAt: monday}.Restore()
	require.NoError(t, err)
	return name
}

func TestTheNameLeftIsKeptOnceHoweverOftenItIsHeard(t *testing.T) {
	profiles, kept := inmemory_player_store.New(), inmemory_past_name_store.New()
	require.NoError(t, profiles.SaveProfile(t.Context(), players.NewProfile(ada, "Ada", monday)))
	keep := keep_past_name_usecase.New(profiles, kept)

	require.NoError(t, keep.Execute(t.Context(), leftByAda(t)))
	require.NoError(t, keep.Execute(t.Context(), leftByAda(t)))

	assert.Equal(t, []pastnames.Snapshot{{Account: ada, Name: "BraveFox42", LeftAt: monday}}, kept.Kept())
}

func TestAnAccountForgottenSinceKeepsNothing(t *testing.T) {
	kept := inmemory_past_name_store.New()

	require.NoError(t, keep_past_name_usecase.New(inmemory_player_store.New(), kept).Execute(t.Context(), leftByAda(t)))

	assert.Empty(t, kept.Kept())
}

func TestAFailureIsAnError(t *testing.T) {
	profiles := inmemory_player_store.New()
	require.NoError(t, profiles.SaveProfile(t.Context(), players.NewProfile(ada, "Ada", monday)))
	profiles.FailWith(errors.New("postgres is down"))
	require.Error(t, keep_past_name_usecase.New(profiles, inmemory_past_name_store.New()).Execute(t.Context(), leftByAda(t)),
		"a failed read is not an account forgotten")
}
