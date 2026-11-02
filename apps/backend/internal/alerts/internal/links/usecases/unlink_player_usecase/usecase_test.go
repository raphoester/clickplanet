package unlink_player_usecase_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/discord"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/links"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/links/usecases/unlink_player_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/roles"
)

var (
	now     = time.Date(2026, 11, 2, 18, 0, 0, 0, time.UTC)
	account = links.AccountID(uuid.MustParse("01a19416-ad77-72b3-8c34-8bbbaec97d8b"))
)

type fakeMembers struct {
	revoked []string
	err     error
}

func (m *fakeMembers) Revoke(_ context.Context, user discord.UserID, role discord.RoleID) error {
	if m.err != nil {
		return m.err
	}
	m.revoked = append(m.revoked, string(user)+" "+string(role))
	return nil
}

func setUp(t *testing.T) (*links.InMemoryStore, *fakeMembers, *unlink_player_usecase.UseCase) {
	t.Helper()
	store := links.NewInMemoryStore()
	members := &fakeMembers{}
	book := roles.NewBook(roles.NewInMemoryStore(), &roles.FakeMaker{})
	_, err := book.Ensure(t.Context(), roles.Nation("ie"))
	require.NoError(t, err)
	return store, members, unlink_player_usecase.New(store, book, members)
}

func TestAnUnlinkedPlayerLosesItsLinkAndItsFlagsRole(t *testing.T) {
	store, members, useCase := setUp(t)
	require.NoError(t, store.SaveLinks(t.Context(), links.NewLink(account, "4242", "ie", now)))

	unlinked, err := useCase.Execute(t.Context(), account)
	require.NoError(t, err)

	assert.True(t, unlinked)
	assert.Equal(t, []string{"4242 1"}, members.revoked)
	assert.Empty(t, store.Links())
}

func TestAnAccountNotLinkedIsNotUnlinked(t *testing.T) {
	_, members, useCase := setUp(t)

	unlinked, err := useCase.Execute(t.Context(), account)
	require.NoError(t, err)

	assert.False(t, unlinked)
	assert.Empty(t, members.revoked)
}

func TestAUserWhoLeftTheServerIsUnlinkedStill(t *testing.T) {
	store, members, useCase := setUp(t)
	members.err = discord.ErrNotMember
	require.NoError(t, store.SaveLinks(t.Context(), links.NewLink(account, "4242", "ie", now)))

	unlinked, err := useCase.Execute(t.Context(), account)
	require.NoError(t, err)

	assert.True(t, unlinked)
	assert.Empty(t, store.Links())
}

func TestARoleDiscordWouldNotTakeOffKeepsTheLink(t *testing.T) {
	store, members, useCase := setUp(t)
	members.err = discord.ErrRefused
	require.NoError(t, store.SaveLinks(t.Context(), links.NewLink(account, "4242", "ie", now)))

	_, err := useCase.Execute(t.Context(), account)
	require.ErrorIs(t, err, discord.ErrRefused)

	assert.Len(t, store.Links(), 1)
}
