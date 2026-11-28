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
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/nations"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/nations/postgres_sold_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/pings"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/pings/postgres_ping_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/reviews"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/reviews/postgres_review_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/roles"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/roles/postgres_role_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg/cppgtest"
)

var (
	now    = time.Date(2026, 11, 2, 18, 0, 0, 123000, time.UTC)
	first  = links.AccountID(uuid.MustParse("01a19416-ad77-72b3-8c34-8bbbaec97d8b"))
	second = links.AccountID(uuid.MustParse("01a193de-df90-7637-9d85-7256584024a6"))
)

func TestTheStoresKeepWhatTheyAreGiven(t *testing.T) {
	db := cppgtest.StartServer(t).OpenSchema(t, "alerts", migrations.FS)

	t.Run("a link, by account and by user, with or without a flag and a name", func(t *testing.T) {
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

		readBefore := link
		link.Rename("BraveFox42")
		require.NoError(t, store.SaveName(t.Context(), link))
		readBefore.Follow("gb")
		require.NoError(t, store.SaveLinks(t.Context(), readBefore))
		byAccount, err = store.LinkOf(t.Context(), first)
		require.NoError(t, err)
		assert.Equal(t, nations.Flag("gb"), byAccount.TakeSnapshot().Flag, "the flag's move is kept")
		assert.Equal(t, links.Name("BraveFox42"), byAccount.TakeSnapshot().Name, "the name given meanwhile is kept")
		link = byAccount

		link.Rename("")
		require.NoError(t, store.SaveName(t.Context(), link))
		byAccount, err = store.LinkOf(t.Context(), first)
		require.NoError(t, err)
		assert.Equal(t, link.TakeSnapshot(), byAccount.TakeSnapshot())

		link.Rename("Ada")
		require.NoError(t, store.SaveName(t.Context(), link))

		link.Forget()
		moved := links.NewLink(second, "4242", "gb", now)
		require.NoError(t, store.SaveLinks(t.Context(), moved, link))
		_, err = store.LinkOf(t.Context(), first)
		require.ErrorIs(t, err, links.ErrNoLink)
		byUser, err = store.LinkOfUser(t.Context(), "4242")
		require.NoError(t, err)
		assert.Equal(t, moved.TakeSnapshot(), byUser.TakeSnapshot(), "another account's link starts with no name")
	})

	t.Run("a role by its key, with its color, until it is forgotten", func(t *testing.T) {
		store := postgres_role_store.New(db)
		ireland := roles.Nation("ie").Key
		_, err := store.RoleOf(t.Context(), ireland)
		require.ErrorIs(t, err, roles.ErrNoRole)

		require.NoError(t, store.SaveRole(t.Context(), ireland, roles.Kept{ID: "77"}))
		require.NoError(t, store.SaveRole(t.Context(), "battles", roles.Kept{ID: "78"}))
		require.NoError(t, store.SaveRole(t.Context(), ireland, roles.Kept{ID: "77", Color: 0x169b62}))
		kept, err := store.RoleOf(t.Context(), ireland)
		require.NoError(t, err)
		assert.Equal(t, roles.Kept{ID: "77", Color: 0x169b62}, kept)

		empire := roles.Nation("r-jz9cwqzp0k").Key
		require.NoError(t, store.SaveRole(t.Context(), empire, roles.Kept{ID: "79", Color: 0x123456, Name: "Holy Roman Empire"}))
		kept, err = store.RoleOf(t.Context(), empire)
		require.NoError(t, err)
		assert.Equal(t, roles.Kept{ID: "79", Color: 0x123456, Name: "Holy Roman Empire"}, kept, "a sold flag's role keeps its name")
		require.NoError(t, store.ForgetRole(t.Context(), empire))

		keys, err := store.Keys(t.Context())
		require.NoError(t, err)
		assert.Equal(t, []roles.Key{"battles", ireland}, keys)
		require.NoError(t, store.ForgetRole(t.Context(), "battles"))
		_, err = store.RoleOf(t.Context(), "battles")
		assert.ErrorIs(t, err, roles.ErrNoRole)
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

	t.Run("the sold flags heard, by their latest name", func(t *testing.T) {
		store := postgres_sold_store.New(db)
		catalog, err := nations.Load(t.Context(), store)
		require.NoError(t, err)
		_, ok := catalog.FlagOf("r-jz9cwqzp0k")
		require.False(t, ok)

		_, _, err = catalog.Name(t.Context(), "r-jz9cwqzp0k", "Holy Roman Empire", now)
		require.NoError(t, err)
		_, _, err = catalog.Name(t.Context(), "gb-eng", "England", now)
		require.NoError(t, err)
		_, _, err = catalog.Name(t.Context(), "r-jz9cwqzp0k", "Heiliges Römisches Reich", now.Add(time.Hour))
		require.NoError(t, err)

		reloaded, err := nations.Load(t.Context(), store)
		require.NoError(t, err)
		_, ok = reloaded.FlagOf("r-jz9cwqzp0k")
		assert.True(t, ok)
		assert.Equal(t, "Heiliges Römisches Reich", reloaded.NameOf("r-jz9cwqzp0k"))
		assert.Equal(t, "England", reloaded.NameOf("gb-eng"))
		sold, err := store.SoldFlags(t.Context())
		require.NoError(t, err)
		assert.Equal(t, []nations.SoldSnapshot{
			{Flag: "gb-eng", Name: "England", NamedAt: now},
			{Flag: "r-jz9cwqzp0k", Name: "Heiliges Römisches Reich", NamedAt: now.Add(time.Hour)},
		}, []nations.SoldSnapshot{sold[0].TakeSnapshot(), sold[1].TakeSnapshot()})
	})

	t.Run("the reviews told", func(t *testing.T) {
		store := postgres_review_store.New(db)
		const smurfs = reviews.SubmissionID("00000000-0000-0000-0000-000000000001")
		told, err := store.Told(t.Context(), smurfs, reviews.Appeal)
		require.NoError(t, err)
		require.False(t, told)

		require.NoError(t, store.SaveTold(t.Context(), smurfs, reviews.Unsure, now))
		require.NoError(t, store.SaveTold(t.Context(), smurfs, reviews.Unsure, now), "kept once")
		told, err = store.Told(t.Context(), smurfs, reviews.Unsure)
		require.NoError(t, err)
		assert.True(t, told)
		told, err = store.Told(t.Context(), smurfs, reviews.Appeal)
		require.NoError(t, err)
		assert.False(t, told, "each cause is told once")
	})
}
