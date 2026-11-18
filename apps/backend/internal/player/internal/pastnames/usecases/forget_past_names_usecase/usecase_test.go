package forget_past_names_usecase_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/pastnames"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/pastnames/inmemory_past_name_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/pastnames/usecases/forget_past_names_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
)

func TestTheNamesItsPlayerLeftAreForgottenWithTheAccountAlone(t *testing.T) {
	store := inmemory_past_name_store.New()
	gone, kept := players.AccountID{15: 1}, players.AccountID{15: 2}
	for _, account := range []players.AccountID{gone, kept} {
		name, err := pastnames.Snapshot{Account: account, Name: "BraveFox42", LeftAt: time.Now()}.Restore()
		require.NoError(t, err)
		require.NoError(t, store.Record(t.Context(), name))
	}

	forget := forget_past_names_usecase.New(store)
	require.NoError(t, forget.Execute(t.Context(), gone))
	require.NoError(t, forget.Execute(t.Context(), gone), "told twice, it deletes nothing more")

	require.Len(t, store.Kept(), 1)
	assert.Equal(t, kept, store.Kept()[0].Account)
}

func TestAFailureIsAnError(t *testing.T) {
	store := inmemory_past_name_store.New()
	store.FailWith(errors.New("postgres is down"))

	assert.Error(t, forget_past_names_usecase.New(store).Execute(t.Context(), players.AccountID{15: 1}))
}
