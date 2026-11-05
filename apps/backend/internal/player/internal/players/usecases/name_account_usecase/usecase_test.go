package name_account_usecase_test

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
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/usecases/name_account_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var (
	now = time.Date(2026, 10, 4, 12, 0, 0, 0, time.UTC)
	ada = players.AccountID{15: 1}
)

func useCase(store *inmemory_player_store.Store, outbox *cpatomic.FakeOutbox) *name_account_usecase.UseCase {
	return name_account_usecase.New(cpatomic.FakeTransactioner{},
		players.NewGeneratedNames(store, players.NewRepeatedNames("BraveFox42")), store, outbox, cptime.NewFixedClock(now))
}

func TestANamelessAccountIsGivenAGeneratedNameAndBecomesAPlayer(t *testing.T) {
	store, outbox := inmemory_player_store.New(), &cpatomic.FakeOutbox{}

	profile, err := useCase(store, outbox).Execute(t.Context(), ada, "")

	require.NoError(t, err)
	assert.Equal(t, players.Name("BraveFox42"), profile.TakeSnapshot().Name)
	assert.True(t, now.Equal(profile.TakeSnapshot().UpdatedAt))
	require.Len(t, outbox.Recorded(), 1)
	assert.True(t, proto.Equal(&playerv1.PlayerCreated{AccountId: ada.String()}, outbox.Recorded()[0]))
}

func TestAnAccountWithANameKeepsItAndIsNoNewPlayer(t *testing.T) {
	store, outbox := inmemory_player_store.New(), &cpatomic.FakeOutbox{}
	require.NoError(t, store.SaveProfile(t.Context(), players.NewProfile(ada, "Ada", now)))

	profile, err := useCase(store, outbox).Execute(t.Context(), ada, "")

	require.NoError(t, err)
	assert.Equal(t, players.Name("Ada"), profile.TakeSnapshot().Name)
	assert.Empty(t, outbox.Recorded())
}

func TestAFailureToNameIsAnErrorAndRecordsNothing(t *testing.T) {
	store, outbox := inmemory_player_store.New(), &cpatomic.FakeOutbox{}
	refused := errors.New("postgres is down")
	store.FailWith(refused)

	_, err := useCase(store, outbox).Execute(t.Context(), ada, "")

	assert.ErrorIs(t, err, refused)
	assert.Empty(t, outbox.Recorded())
}
