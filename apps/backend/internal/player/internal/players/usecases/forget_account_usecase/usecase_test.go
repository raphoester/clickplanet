package forget_account_usecase_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/inmemory_player_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/usecases/forget_account_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
)

func TestTheProfileIsForgottenAndTheOtherModulesAreTold(t *testing.T) {
	store, outbox := inmemory_player_store.New(), &cpatomic.FakeOutbox{}
	gone, kept := players.AccountID{15: 1}, players.AccountID{15: 2}
	for name, account := range map[players.Name]players.AccountID{"Ada": gone, "Bob": kept} {
		require.NoError(t, store.SaveProfile(t.Context(), players.NewProfile(account, name, time.Now())))
	}

	require.NoError(t, forget_account_usecase.New(cpatomic.FakeTransactioner{}, store, outbox).Execute(t.Context(), gone))

	_, err := store.Profile(t.Context(), gone)
	require.ErrorIs(t, err, players.ErrNoProfile)
	_, err = store.Profile(t.Context(), kept)
	require.NoError(t, err)
	require.Len(t, outbox.Recorded(), 1)
	assert.True(t, proto.Equal(&playerv1.PlayerForgotten{AccountId: gone.String()}, outbox.Recorded()[0]))
}

func TestAnAccountPlayerNeverKnewIsToldForgottenToo(t *testing.T) {
	outbox := &cpatomic.FakeOutbox{}

	require.NoError(t, forget_account_usecase.New(cpatomic.FakeTransactioner{}, inmemory_player_store.New(), outbox).
		Execute(t.Context(), players.AccountID{15: 3}))

	require.Len(t, outbox.Recorded(), 1, "another module may keep its stats")
}

func TestAStoreFailureIsAnErrorAndTellsNothing(t *testing.T) {
	store, outbox := inmemory_player_store.New(), &cpatomic.FakeOutbox{}
	store.FailWith(errors.New("postgres is down"))

	require.Error(t, forget_account_usecase.New(cpatomic.FakeTransactioner{}, store, outbox).Execute(t.Context(), players.AccountID{15: 1}))
	assert.Empty(t, outbox.Recorded())
}
