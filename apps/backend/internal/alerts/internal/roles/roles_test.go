package roles_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/discord"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/roles"
)

func TestARoleIsMadeOnceAndFoundAfter(t *testing.T) {
	maker := &roles.FakeMaker{}
	book := roles.NewBook(roles.NewInMemoryStore(), maker)

	_, err := book.IDOf(t.Context(), roles.Nation("ie"))
	require.ErrorIs(t, err, roles.ErrNoRole)

	first, err := book.Ensure(t.Context(), roles.Nation("ie"))
	require.NoError(t, err)
	again, err := book.Ensure(t.Context(), roles.Nation("ie"))
	require.NoError(t, err)
	battles, err := book.Ensure(t.Context(), roles.Battles())
	require.NoError(t, err)
	found, err := book.IDOf(t.Context(), roles.Nation("ie"))
	require.NoError(t, err)

	assert.Equal(t, discord.RoleID("1"), first)
	assert.Equal(t, first, again)
	assert.Equal(t, first, found)
	assert.Equal(t, discord.RoleID("2"), battles)
	assert.Equal(t, []string{"Ireland", "Battles"}, maker.Made)
}
