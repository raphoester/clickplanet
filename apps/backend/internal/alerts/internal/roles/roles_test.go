package roles_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/discord"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/nations"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/roles"
)

func TestARoleIsMadeOnceInItsColorAndFoundAfter(t *testing.T) {
	maker := &roles.FakeMaker{}
	book := roles.NewBook(roles.NewInMemoryStore(), maker)

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
	for _, flag := range []nations.Flag{"fr", "de", "gb", "us", "ie", "it", "es", "br", "jp", "xb"} {
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
	book := roles.NewBook(store, maker)

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
	book := roles.NewBook(store, maker)

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
	book := roles.NewBook(store, maker)

	require.NoError(t, book.Tidy(t.Context()))

	assert.Equal(t, []discord.RoleID{"70"}, maker.Deleted)
	assert.Equal(t, map[discord.RoleID]discord.Color{"71": roles.ColorOf("fr")}, maker.Painted)
	keys, err := store.Keys(t.Context())
	require.NoError(t, err)
	assert.Equal(t, []roles.Key{roles.Nation("fr").Key, roles.Nation("ie").Key}, keys)
}
