package rename_player_usecase_test

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
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/usecases/rename_player_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var now = time.Date(2026, 11, 3, 12, 0, 0, 0, time.UTC)

var (
	ada = players.AccountID{15: 1}
	bob = players.AccountID{15: 2}
)

func chosenAnHourAgo(t *testing.T, store *inmemory_player_store.Store, account players.AccountID, name players.Name) {
	t.Helper()

	snapshot := players.ProfileSnapshot{Account: account, Name: name, UpdatedAt: now.Add(-time.Hour), ChosenAt: now.Add(-time.Hour)}
	require.NoError(t, store.SaveProfile(t.Context(), snapshot.Restore()))
}

func TestANameChosenAnHourAgoIsReplacedAndItsWaitLifted(t *testing.T) {
	store := inmemory_player_store.New()
	chosenAnHourAgo(t, store, ada, "Rubbish")

	profile, err := rename_player_usecase.New(cpatomic.FakeTransactioner{}, store, &cpatomic.FakeOutbox{}, cptime.NewFixedClock(now)).
		Execute(t.Context(), rename_player_usecase.In{Account: ada, Name: " Ada_L "})

	require.NoError(t, err)
	want := players.ProfileSnapshot{Account: ada, Name: "Ada_L", UpdatedAt: now}.Restore()
	assert.Equal(t, want, profile)
	stored, err := store.Profile(t.Context(), ada)
	require.NoError(t, err)
	assert.Equal(t, want, stored)
	_, err = stored.Renamed("Ada", now)
	assert.NoError(t, err, "the player's next choice goes at once")
}

func TestAnInvalidNameChangesNothing(t *testing.T) {
	store := inmemory_player_store.New()
	chosenAnHourAgo(t, store, ada, "Ada")

	_, err := rename_player_usecase.New(cpatomic.FakeTransactioner{}, store, &cpatomic.FakeOutbox{}, cptime.NewFixedClock(now)).
		Execute(t.Context(), rename_player_usecase.In{Account: ada, Name: "guest_ada"})

	require.ErrorIs(t, err, players.ErrInvalidName)
	stored, err := store.Profile(t.Context(), ada)
	require.NoError(t, err)
	assert.Equal(t, players.Name("Ada"), stored.TakeSnapshot().Name)
}

func TestAnAccountWithNoUsernameIsNotGivenOne(t *testing.T) {
	store := inmemory_player_store.New()

	_, err := rename_player_usecase.New(cpatomic.FakeTransactioner{}, store, &cpatomic.FakeOutbox{}, cptime.NewFixedClock(now)).
		Execute(t.Context(), rename_player_usecase.In{Account: ada, Name: "Ada"})

	require.ErrorIs(t, err, players.ErrNoProfile)
	_, err = store.Profile(t.Context(), ada)
	assert.ErrorIs(t, err, players.ErrNoProfile)
}

func TestANameAnotherPlayerHoldsIsTaken(t *testing.T) {
	store := inmemory_player_store.New()
	chosenAnHourAgo(t, store, ada, "Ada")
	chosenAnHourAgo(t, store, bob, "Bob")

	_, err := rename_player_usecase.New(cpatomic.FakeTransactioner{}, store, &cpatomic.FakeOutbox{}, cptime.NewFixedClock(now)).
		Execute(t.Context(), rename_player_usecase.In{Account: bob, Name: "ADA"})

	require.ErrorIs(t, err, players.ErrNameTaken)
}

func TestAStoreFailureIsNotAMissingProfile(t *testing.T) {
	store := inmemory_player_store.New()
	store.FailWith(errors.New("postgres is down"))

	_, err := rename_player_usecase.New(cpatomic.FakeTransactioner{}, store, &cpatomic.FakeOutbox{}, cptime.NewFixedClock(now)).
		Execute(t.Context(), rename_player_usecase.In{Account: ada, Name: "Ada"})

	require.Error(t, err)
	assert.NotErrorIs(t, err, players.ErrNoProfile)
	assert.NotErrorIs(t, err, players.ErrNameTaken)
}

func TestARenameIsTold(t *testing.T) {
	store, outbox := inmemory_player_store.New(), &cpatomic.FakeOutbox{}
	chosenAnHourAgo(t, store, ada, "Rubbish")

	_, err := rename_player_usecase.New(cpatomic.FakeTransactioner{}, store, outbox, cptime.NewFixedClock(now)).
		Execute(t.Context(), rename_player_usecase.In{Account: ada, Name: "Ada_L"})

	require.NoError(t, err)
	require.Len(t, outbox.Recorded(), 1)
	assert.True(t, proto.Equal(&playerv1.PlayerNamed{AccountId: ada.String()}, outbox.Recorded()[0]))
}

func TestARenameRefusedIsNotTold(t *testing.T) {
	store, outbox := inmemory_player_store.New(), &cpatomic.FakeOutbox{}
	chosenAnHourAgo(t, store, ada, "Ada")
	chosenAnHourAgo(t, store, bob, "Bob")

	_, err := rename_player_usecase.New(cpatomic.FakeTransactioner{}, store, outbox, cptime.NewFixedClock(now)).
		Execute(t.Context(), rename_player_usecase.In{Account: bob, Name: "ADA"})

	require.ErrorIs(t, err, players.ErrNameTaken)
	assert.Empty(t, outbox.Recorded())
}
