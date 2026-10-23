package allegiances_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/allegiances"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcountries"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

const (
	alice = "01926c6e-7a4b-7c3d-8e9f-0a1b2c3d4e5f"
	bob   = "01926c6e-7a4b-7c3d-8e9f-0a1b2c3d4e60"
)

func book() (allegiances.Book, *allegiances.MemoryStore, *cptime.FixedClock) {
	store := allegiances.NewMemoryStore()
	clock := cptime.NewFixedClock(at)

	return allegiances.NewBook(store, cpcountries.New(), clock), store, clock
}

func TestAnAccountIsHeldToItsFlagForAnHourAfterAChange(t *testing.T) {
	b, store, clock := book()
	payer := clicks.Payer{Account: alice}

	require.NoError(t, b.Pledge(t.Context(), payer, "fr"))
	require.NoError(t, b.Pledge(t.Context(), payer, "de"))

	clock.Advance(59 * time.Minute)
	err := b.Pledge(t.Context(), payer, "fr")
	require.ErrorIs(t, err, allegiances.ErrChangedTooSoon)
	assert.Equal(t, allegiances.Snapshot{Flag: "de", ChangedAt: at}, store.Kept(alice))

	clock.Advance(time.Minute)
	require.NoError(t, b.Pledge(t.Context(), payer, "fr"))
	assert.Equal(t, allegiances.Snapshot{Flag: "fr", ChangedAt: at.Add(time.Hour)}, store.Kept(alice))
}

func TestEachAccountHasItsOwnFlag(t *testing.T) {
	b, store, _ := book()

	require.NoError(t, b.Pledge(t.Context(), clicks.Payer{Account: alice, Scope: "1.2.3.4"}, "fr"))
	require.NoError(t, b.Pledge(t.Context(), clicks.Payer{Account: alice, Scope: "1.2.3.4"}, "de"))
	require.NoError(t, b.Pledge(t.Context(), clicks.Payer{Account: bob, Scope: "1.2.3.4"}, "fr"))

	assert.Equal(t, "de", store.Kept(alice).Flag)
	assert.Equal(t, "fr", store.Kept(bob).Flag)
}

func TestACallerWithNoAccountIsHeldToNothing(t *testing.T) {
	b, store, _ := book()

	require.NoError(t, b.Pledge(t.Context(), clicks.Payer{Scope: "1.2.3.4"}, "fr"))
	require.NoError(t, b.Pledge(t.Context(), clicks.Payer{Scope: "1.2.3.4"}, "de"))

	assert.Equal(t, allegiances.Snapshot{}, store.Kept(allegiances.NoAccount))
}

func TestAFlagThatIsNoCountryIsNotKept(t *testing.T) {
	b, store, _ := book()
	payer := clicks.Payer{Account: alice}
	require.NoError(t, b.Pledge(t.Context(), payer, "fr"))
	require.NoError(t, b.Pledge(t.Context(), payer, "de"))

	require.NoError(t, b.Pledge(t.Context(), payer, "zz"))

	assert.Equal(t, "de", store.Kept(alice).Flag)
}

func TestAStoreThatFailsFailsThePledge(t *testing.T) {
	b, store, _ := book()
	down := errors.New("postgres is down")
	store.FailWith(down)

	assert.ErrorIs(t, b.Pledge(t.Context(), clicks.Payer{Account: alice}, "fr"), down)
}
