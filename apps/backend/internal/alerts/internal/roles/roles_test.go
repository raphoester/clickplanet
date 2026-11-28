package roles_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/discord"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/nations"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/roles"
)

func TestARoleIsMadeOnceInItsColorAndFoundAfter(t *testing.T) {
	maker := &roles.FakeMaker{}
	book := roles.NewBook(roles.NewInMemoryStore(), maker, nations.NewCatalog(nil))

	_, err := book.IDOf(t.Context(), roles.Nation("ie"))
	require.ErrorIs(t, err, roles.ErrNoRole)

	first, err := book.Ensure(t.Context(), roles.Nation("ie"))
	require.NoError(t, err)
	again, err := book.Ensure(t.Context(), roles.Nation("ie"))
	require.NoError(t, err)
	found, err := book.IDOf(t.Context(), roles.Nation("ie"))
	require.NoError(t, err)

	assert.Equal(t, discord.RoleID("1"), first)
	assert.Equal(t, first, again)
	assert.Equal(t, first, found)
	assert.Equal(t, []string{"Ireland"}, maker.Made)
	assert.Equal(t, roles.ColorOf("ie"), maker.Painted["1"])
}

func TestEveryFlagHasABrightColorOfItsOwn(t *testing.T) {
	seen := map[discord.Color]nations.Flag{}
	for _, flag := range []nations.Flag{"fr", "de", "gb", "us", "ie", "it", "es", "br", "jp", "pl"} {
		color := roles.ColorOf(flag)
		r, g, b := int(color>>16)&0xff, int(color>>8)&0xff, int(color)&0xff

		assert.Equal(t, color, roles.ColorOf(flag), "%s: the same on every call", flag)
		assert.Greater(t, max(r, g, b)-min(r, g, b), 100, "%s: %06x is no gray", flag, int(color))
		assert.NotContains(t, seen, color, "%s: %06x is %s's too", flag, int(color), seen[color])
		seen[color] = flag
	}
}

func TestARoleKeptInAnotherColorIsPaintedOnce(t *testing.T) {
	store := roles.NewInMemoryStore()
	require.NoError(t, store.SaveRole(t.Context(), roles.Nation("fr").Key, roles.Kept{ID: "77"}))
	maker := &roles.FakeMaker{}
	book := roles.NewBook(store, maker, nations.NewCatalog(nil))

	id, err := book.Ensure(t.Context(), roles.Nation("fr"))
	require.NoError(t, err)
	_, err = book.Ensure(t.Context(), roles.Nation("fr"))
	require.NoError(t, err)

	assert.Equal(t, discord.RoleID("77"), id)
	assert.Equal(t, map[discord.RoleID]discord.Color{"77": roles.ColorOf("fr")}, maker.Painted)
	assert.Empty(t, maker.Made)
}

func TestARoleDeletedByHandIsMadeAgain(t *testing.T) {
	store := roles.NewInMemoryStore()
	require.NoError(t, store.SaveRole(t.Context(), roles.Nation("fr").Key, roles.Kept{ID: "77"}))
	maker := &roles.FakeMaker{Gone: map[discord.RoleID]bool{"77": true}}
	book := roles.NewBook(store, maker, nations.NewCatalog(nil))

	id, err := book.Ensure(t.Context(), roles.Nation("fr"))
	require.NoError(t, err)

	assert.Equal(t, discord.RoleID("1"), id)
	assert.Equal(t, []string{"France"}, maker.Made)
}

func TestTidyingPaintsTheFlagsAndDeletesTheRolesTheGameNoLongerNames(t *testing.T) {
	store := roles.NewInMemoryStore()
	require.NoError(t, store.SaveRole(t.Context(), "battles", roles.Kept{ID: "70"}))
	require.NoError(t, store.SaveRole(t.Context(), roles.Nation("fr").Key, roles.Kept{ID: "71"}))
	require.NoError(t, store.SaveRole(t.Context(), roles.Nation("ie").Key, roles.Kept{ID: "72", Color: roles.ColorOf("ie")}))
	maker := &roles.FakeMaker{}
	book := roles.NewBook(store, maker, nations.NewCatalog(nil))

	require.NoError(t, book.Tidy(t.Context()))

	assert.Equal(t, []discord.RoleID{"70"}, maker.Deleted)
	assert.Equal(t, map[discord.RoleID]discord.Color{"71": roles.ColorOf("fr")}, maker.Painted)
	keys, err := store.Keys(t.Context())
	require.NoError(t, err)
	assert.Equal(t, []roles.Key{roles.Nation("fr").Key, roles.Nation("ie").Key}, keys)
}

func TestASoldFlagsRoleWearsItsNameAndFollowsARename(t *testing.T) {
	const empire = nations.Flag("r-jz9cwqzp0k")
	store := roles.NewInMemoryStore()
	maker := &roles.FakeMaker{}
	catalog, err := nations.Load(t.Context(), nations.NewInMemoryStore())
	require.NoError(t, err)
	_, _, err = catalog.Name(t.Context(), string(empire), "Holy Roman Empire", time.Time{})
	require.NoError(t, err)
	book := roles.NewBook(store, maker, catalog)

	require.NoError(t, book.Refresh(t.Context(), roles.Nation(empire)))
	assert.Empty(t, maker.Made, "no role made before a grant needs one")

	id, err := book.Ensure(t.Context(), roles.Nation(empire))
	require.NoError(t, err)
	assert.Equal(t, []string{"Holy Roman Empire"}, maker.Made)

	_, _, err = catalog.Name(t.Context(), string(empire), "Heiliges Römisches Reich", time.Unix(1, 0))
	require.NoError(t, err)
	require.NoError(t, book.Refresh(t.Context(), roles.Nation(empire)))
	require.NoError(t, book.Refresh(t.Context(), roles.Nation(empire)))

	assert.Equal(t, map[discord.RoleID]string{id: "Heiliges Römisches Reich"}, maker.Renamed, "renamed once")
	assert.Equal(t, map[discord.RoleID]discord.Color{id: roles.ColorOf(empire)}, maker.Painted, "painted at its making alone")
	kept, err := store.RoleOf(t.Context(), roles.Nation(empire).Key)
	require.NoError(t, err)
	assert.Equal(t, "Heiliges Römisches Reich", kept.Name)
}

func TestAFlagTheGameShipsWithKeepsTheNameItWasMadeWith(t *testing.T) {
	store := roles.NewInMemoryStore()
	require.NoError(t, store.SaveRole(t.Context(), roles.Nation("ie").Key, roles.Kept{ID: "72", Color: roles.ColorOf("ie")}))
	maker := &roles.FakeMaker{}

	require.NoError(t, roles.NewBook(store, maker, nations.NewCatalog(nil)).Refresh(t.Context(), roles.Nation("ie")))

	assert.Empty(t, maker.Renamed)
	assert.Empty(t, maker.Painted)
}

func TestTidyingKeepsTheSoldFlagsHeardAndDeletesTheOthers(t *testing.T) {
	store := roles.NewInMemoryStore()
	require.NoError(t, store.SaveRole(t.Context(), roles.Nation("r-jz9cwqzp0k").Key,
		roles.Kept{ID: "80", Color: roles.ColorOf("r-jz9cwqzp0k"), Name: "Holy Roman Empire"}))
	require.NoError(t, store.SaveRole(t.Context(), roles.Nation("xb").Key, roles.Kept{ID: "81", Color: roles.ColorOf("xb")}))
	maker := &roles.FakeMaker{}
	book := roles.NewBook(store, maker, nations.NewCatalog(map[nations.Flag]string{"r-jz9cwqzp0k": "Heiliges Römisches Reich"}))

	require.NoError(t, book.Tidy(t.Context()))

	assert.Equal(t, []discord.RoleID{"81"}, maker.Deleted, "a flag that left the build and was never heard sold")
	assert.Equal(t, map[discord.RoleID]string{"80": "Heiliges Römisches Reich"}, maker.Renamed)
}
