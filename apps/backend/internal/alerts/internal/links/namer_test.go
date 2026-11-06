package links_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/discord"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/links"
)

func namer(t *testing.T) (*links.InMemoryStore, *links.FakeNames, *links.FakeRenamer, links.Namer) {
	t.Helper()
	store, names, members := links.NewInMemoryStore(), links.NewFakeNames(), &links.FakeRenamer{}
	return store, names, members, links.NewNamer(store, names, members)
}

func TestALinkedUserIsGivenItsAccountsUsernameOnce(t *testing.T) {
	store, names, members, namer := namer(t)
	require.NoError(t, store.SaveLinks(t.Context(), links.NewLink(account, "42", "ie", now)))
	names.Name(account, "BraveFox42")

	renamed, err := namer.Name(t.Context(), account)
	require.NoError(t, err)
	assert.True(t, renamed)
	renamed, err = namer.Name(t.Context(), account)
	require.NoError(t, err)
	assert.False(t, renamed, "it wears it already")

	assert.Equal(t, []string{"42 BraveFox42"}, members.Renamed())
	assert.Equal(t, []links.LinkSnapshot{{Account: account, User: "42", Flag: "ie", Name: "BraveFox42", LinkedAt: now}},
		store.Links())
}

func TestARenameIsFollowed(t *testing.T) {
	store, names, members, namer := namer(t)
	require.NoError(t, store.SaveLinks(t.Context(), links.NewLink(account, "42", "ie", now)))
	names.Name(account, "BraveFox42")
	_, err := namer.Name(t.Context(), account)
	require.NoError(t, err)

	names.Name(account, "Ada_L")
	renamed, err := namer.Name(t.Context(), account)

	require.NoError(t, err)
	assert.True(t, renamed)
	assert.Equal(t, []string{"42 BraveFox42", "42 Ada_L"}, members.Renamed())
}

func TestAnAccountPlayerNoLongerShowsLosesItsNickname(t *testing.T) {
	store, names, members, namer := namer(t)
	require.NoError(t, store.SaveLinks(t.Context(), links.NewLink(account, "42", "ie", now)))
	names.Name(account, "BraveFox42")
	_, err := namer.Name(t.Context(), account)
	require.NoError(t, err)

	names.Name(account, "")
	renamed, err := namer.Name(t.Context(), account)

	require.NoError(t, err)
	assert.True(t, renamed)
	assert.Equal(t, []string{"42 BraveFox42", "42 "}, members.Renamed())
	assert.Empty(t, store.Links()[0].Name)
}

func TestAnAccountWithNoLinkIsNamedNowhere(t *testing.T) {
	_, names, members, namer := namer(t)
	names.Name(account, "BraveFox42")

	renamed, err := namer.Name(t.Context(), account)

	require.NoError(t, err)
	assert.False(t, renamed)
	assert.Empty(t, members.Renamed())
}

func TestAUserWhoLeftTheServerIsForgotten(t *testing.T) {
	store, names, members, namer := namer(t)
	require.NoError(t, store.SaveLinks(t.Context(), links.NewLink(account, "42", "ie", now)))
	names.Name(account, "BraveFox42")
	members.Gone = map[discord.UserID]bool{"42": true}

	renamed, err := namer.Name(t.Context(), account)

	require.NoError(t, err)
	assert.False(t, renamed)
	assert.Empty(t, store.Links())
}

func TestAMemberTheBotMayNotRenameKeepsItsLinkUnnamed(t *testing.T) {
	store, names, members, namer := namer(t)
	require.NoError(t, store.SaveLinks(t.Context(), links.NewLink(account, "42", "ie", now)))
	names.Name(account, "BraveFox42")
	members.Refused = map[discord.UserID]bool{"42": true}

	_, err := namer.Name(t.Context(), account)

	require.ErrorIs(t, err, discord.ErrRefused)
	assert.Equal(t, []links.LinkSnapshot{{Account: account, User: "42", Flag: "ie", LinkedAt: now}}, store.Links(),
		"a later call tries again")
}

func TestPlayerUnreachableIsAFailureThatRenamesNobody(t *testing.T) {
	store, names, members, namer := namer(t)
	require.NoError(t, store.SaveLinks(t.Context(), links.NewLink(account, "42", "ie", now)))
	down := errors.New("player is down")
	names.FailWith(down)

	_, err := namer.Name(t.Context(), account)

	require.ErrorIs(t, err, down)
	assert.Empty(t, members.Renamed())
}

func TestANameGivenWhileTheFlagMovedKeepsTheMove(t *testing.T) {
	store := links.NewInMemoryStore()
	read := links.NewLink(account, "42", "ie", now)
	require.NoError(t, store.SaveLinks(t.Context(), read))
	moved := read
	moved.Follow("gb")
	require.NoError(t, store.SaveLinks(t.Context(), moved))

	read.Rename("BraveFox42")
	require.NoError(t, store.SaveName(t.Context(), read))

	assert.Equal(t, []links.LinkSnapshot{{Account: account, User: "42", Flag: "gb", Name: "BraveFox42", LinkedAt: now}},
		store.Links())
}

func TestAFlagMovedWhileANameWasGivenKeepsTheName(t *testing.T) {
	store := links.NewInMemoryStore()
	read := links.NewLink(account, "42", "ie", now)
	require.NoError(t, store.SaveLinks(t.Context(), read))
	named := read
	named.Rename("BraveFox42")
	require.NoError(t, store.SaveName(t.Context(), named))

	read.Follow("gb")
	require.NoError(t, store.SaveLinks(t.Context(), read))

	assert.Equal(t, []links.LinkSnapshot{{Account: account, User: "42", Flag: "gb", Name: "BraveFox42", LinkedAt: now}},
		store.Links())
}
