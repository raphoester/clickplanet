package allegiances_test

import (
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/allegiances"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/allegiances/inmemory_allegiance_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcountries"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

const (
	alice = "01926c6e-7a4b-7c3d-8e9f-0a1b2c3d4e5f"
	bob   = "01926c6e-7a4b-7c3d-8e9f-0a1b2c3d4e60"
)

func book(t *testing.T) (allegiances.Book, *inmemory_allegiance_storage.Storage, *cptime.FixedClock) {
	t.Helper()

	storage := inmemory_allegiance_storage.New(inmemory_allegiance_storage.Config{},
		inmemory_allegiance_storage.NewMemoryPersistence(), slog.New(slog.DiscardHandler))
	clock := cptime.NewFixedClock(at)

	return allegiances.NewBook(storage, cpcountries.New(), clock), storage, clock
}

func TestAnAccountIsHeldToItsFlagForAnHourAfterAChange(t *testing.T) {
	b, storage, clock := book(t)
	payer := clicks.Payer{Account: alice}

	require.NoError(t, b.Pledge(payer, "fr"))
	require.NoError(t, b.Pledge(payer, "de"))

	clock.Advance(59 * time.Minute)
	err := b.Pledge(payer, "fr")
	require.ErrorIs(t, err, allegiances.ErrChangedTooSoon)
	assert.Equal(t, allegiances.Snapshot{Flag: "de", ChangedAt: at}, storage.Snapshot(alice))

	clock.Advance(time.Minute)
	require.NoError(t, b.Pledge(payer, "fr"))
	assert.Equal(t, allegiances.Snapshot{Flag: "fr", ChangedAt: at.Add(time.Hour)}, storage.Snapshot(alice))
}

func TestEachAccountHasItsOwnFlag(t *testing.T) {
	b, storage, _ := book(t)

	require.NoError(t, b.Pledge(clicks.Payer{Account: alice, Scope: "1.2.3.4"}, "fr"))
	require.NoError(t, b.Pledge(clicks.Payer{Account: alice, Scope: "1.2.3.4"}, "de"))
	require.NoError(t, b.Pledge(clicks.Payer{Account: bob, Scope: "1.2.3.4"}, "fr"))

	assert.Equal(t, "de", storage.Snapshot(alice).Flag)
	assert.Equal(t, "fr", storage.Snapshot(bob).Flag)
}

func TestACallerWithNoAccountIsHeldToNothing(t *testing.T) {
	b, storage, _ := book(t)

	require.NoError(t, b.Pledge(clicks.Payer{Scope: "1.2.3.4"}, "fr"))
	require.NoError(t, b.Pledge(clicks.Payer{Scope: "1.2.3.4"}, "de"))

	assert.Equal(t, allegiances.Snapshot{}, storage.Snapshot(allegiances.NoAccount))
}

func TestAFlagThatIsNoCountryIsNotKept(t *testing.T) {
	b, storage, _ := book(t)
	payer := clicks.Payer{Account: alice}
	require.NoError(t, b.Pledge(payer, "fr"))
	require.NoError(t, b.Pledge(payer, "de"))

	require.NoError(t, b.Pledge(payer, "zz"))

	assert.Equal(t, "de", storage.Snapshot(alice).Flag)
}
