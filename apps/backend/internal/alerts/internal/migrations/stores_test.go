package migrations_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/links"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/links/postgres_link_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/pings"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/pings/postgres_ping_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/roles"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/roles/postgres_role_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

var (
	now    = time.Date(2026, 11, 2, 18, 0, 0, 123000, time.UTC)
	first  = links.AccountID(uuid.MustParse("01a19416-ad77-72b3-8c34-8bbbaec97d8b"))
	second = links.AccountID(uuid.MustParse("01a193de-df90-7637-9d85-7256584024a6"))
)

func TestTheStoresKeepWhatTheyAreGiven(t *testing.T) {
	db := cppg.StartTestServer(t).OpenSchema(t, "alerts", migrations.FS)

	t.Run("a flow until it is spent", func(t *testing.T) {
		store := postgres_link_store.New(db)
		_, err := store.FlowOf(t.Context(), first)
		require.ErrorIs(t, err, links.ErrNoFlow)

		flow, err := links.NewFlow(first, &links.SequentialSecrets{}, now)
		require.NoError(t, err)
		require.NoError(t, store.SaveFlow(t.Context(), flow))
		kept, err := store.FlowOf(t.Context(), first)
		require.NoError(t, err)
		assert.Equal(t, flow.TakeSnapshot(), kept.TakeSnapshot())

		_, err = kept.Spend("secret-1", now)
		require.NoError(t, err)
		require.NoError(t, store.SaveFlow(t.Context(), kept))
		_, err = store.FlowOf(t.Context(), first)
		assert.ErrorIs(t, err, links.ErrNoFlow)
	})

	t.Run("a link, by account and by user, with or without a flag", func(t *testing.T) {
		store := postgres_link_store.New(db)
		_, err := store.LinkOf(t.Context(), first)
		require.ErrorIs(t, err, links.ErrNoLink)

		link := links.NewLink(first, "4242", "", now)
		require.NoError(t, store.SaveLinks(t.Context(), link))
		byUser, err := store.LinkOfUser(t.Context(), "4242")
		require.NoError(t, err)
		assert.Equal(t, link.TakeSnapshot(), byUser.TakeSnapshot())

		link.Follow("ie")
		require.NoError(t, store.SaveLinks(t.Context(), link))
		byAccount, err := store.LinkOf(t.Context(), first)
		require.NoError(t, err)
		assert.Equal(t, link.TakeSnapshot(), byAccount.TakeSnapshot())

		link.Forget()
		moved := links.NewLink(second, "4242", "gb", now)
		require.NoError(t, store.SaveLinks(t.Context(), moved, link))
		_, err = store.LinkOf(t.Context(), first)
		require.ErrorIs(t, err, links.ErrNoLink)
		byUser, err = store.LinkOfUser(t.Context(), "4242")
		require.NoError(t, err)
		assert.Equal(t, moved.TakeSnapshot(), byUser.TakeSnapshot())
	})

	t.Run("a role by its key", func(t *testing.T) {
		store := postgres_role_store.New(db)
		_, err := store.RoleOf(t.Context(), roles.Battles().Key)
		require.ErrorIs(t, err, roles.ErrNoRole)

		require.NoError(t, store.SaveRole(t.Context(), roles.Battles().Key, "77"))
		id, err := store.RoleOf(t.Context(), roles.Battles().Key)
		require.NoError(t, err)
		assert.Equal(t, "77", string(id))
	})

	t.Run("the pings", func(t *testing.T) {
		store := postgres_ping_store.New(db)
		require.NoError(t, store.SaveAttackPing(t.Context(), "ie", now.Add(-25*time.Hour)))
		require.NoError(t, store.SaveAttackPing(t.Context(), "ie", now.Add(-time.Hour)))
		require.NoError(t, store.SaveAttackPing(t.Context(), "gb", now))
		history, err := store.AttackPingsOf(t.Context(), "ie", now.Add(-pings.Day))
		require.NoError(t, err)
		assert.Equal(t, pings.History{now.Add(-time.Hour)}, history)

		pinged, err := store.StartPinged(t.Context(), "season-1/battle-1")
		require.NoError(t, err)
		require.False(t, pinged)
		require.NoError(t, store.SaveStartPing(t.Context(), "season-1/battle-1", now))
		pinged, err = store.StartPinged(t.Context(), "season-1/battle-1")
		require.NoError(t, err)
		assert.True(t, pinged)
	})
}
