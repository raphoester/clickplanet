package follow_flag_usecase_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/discord"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/links"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/links/usecases/follow_flag_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/roles"
)

var (
	now     = time.Date(2026, 11, 2, 18, 0, 0, 0, time.UTC)
	account = links.AccountID(uuid.MustParse("01a19416-ad77-72b3-8c34-8bbbaec97d8b"))
)

type fakeMembers struct {
	calls []string
	err   error
}

func (m *fakeMembers) Grant(_ context.Context, user discord.UserID, role discord.RoleID) error {
	if m.err != nil {
		return m.err
	}
	m.calls = append(m.calls, "grant "+string(user)+" "+string(role))
	return nil
}

func (m *fakeMembers) Revoke(_ context.Context, user discord.UserID, role discord.RoleID) error {
	m.calls = append(m.calls, "revoke "+string(user)+" "+string(role))
	return nil
}

func setUp(t *testing.T) (*links.InMemoryStore, *fakeMembers, *follow_flag_usecase.UseCase) {
	t.Helper()
	store := links.NewInMemoryStore()
	members := &fakeMembers{}
	book := roles.NewBook(roles.NewInMemoryStore(), &roles.FakeMaker{})
	_, err := book.Ensure(t.Context(), roles.Nation("ie"))
	require.NoError(t, err)
	return store, members, follow_flag_usecase.New(store, book, members)
}

func TestTheRoleFollowsTheFlagALinkedPlayerPledged(t *testing.T) {
	store, members, useCase := setUp(t)
	require.NoError(t, store.SaveLinks(t.Context(), links.NewLink(account, "4242", "ie", now)))

	require.NoError(t, useCase.Execute(t.Context(), follow_flag_usecase.In{Account: account, Flag: "ie"}))
	require.NoError(t, useCase.Execute(t.Context(), follow_flag_usecase.In{Account: account, Flag: "gb"}))

	assert.Equal(t, []string{"grant 4242 2", "revoke 4242 1"}, members.calls)
	assert.Equal(t, []links.LinkSnapshot{{Account: account, User: "4242", Flag: "gb", LinkedAt: now}}, store.Links())
}

func TestAPlayerWithNoFlagYetIsGrantedItsFirst(t *testing.T) {
	store, members, useCase := setUp(t)
	require.NoError(t, store.SaveLinks(t.Context(), links.NewLink(account, "4242", "", now)))

	require.NoError(t, useCase.Execute(t.Context(), follow_flag_usecase.In{Account: account, Flag: "ie"}))

	assert.Equal(t, []string{"grant 4242 1"}, members.calls)
}

func TestAnAccountWithNoLinkChangesNothing(t *testing.T) {
	_, members, useCase := setUp(t)

	require.NoError(t, useCase.Execute(t.Context(), follow_flag_usecase.In{Account: account, Flag: "gb"}))

	assert.Empty(t, members.calls)
}

func TestAUserWhoLeftTheServerIsForgotten(t *testing.T) {
	store, members, useCase := setUp(t)
	members.err = discord.ErrNotMember
	require.NoError(t, store.SaveLinks(t.Context(), links.NewLink(account, "4242", "ie", now)))

	require.NoError(t, useCase.Execute(t.Context(), follow_flag_usecase.In{Account: account, Flag: "gb"}))

	assert.Empty(t, store.Links())
}

func TestADiscordThatFailsIsTriedAgain(t *testing.T) {
	store, members, useCase := setUp(t)
	members.err = discord.ErrRefused
	require.NoError(t, store.SaveLinks(t.Context(), links.NewLink(account, "4242", "ie", now)))

	require.ErrorIs(t, useCase.Execute(t.Context(), follow_flag_usecase.In{Account: account, Flag: "gb"}), discord.ErrRefused)

	assert.Equal(t, []links.LinkSnapshot{{Account: account, User: "4242", Flag: "ie", LinkedAt: now}}, store.Links())
}
