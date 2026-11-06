package forget_link_usecase_test

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/discord"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/links"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/links/usecases/forget_link_usecase"
)

var (
	now     = time.Date(2026, 11, 2, 18, 0, 0, 0, time.UTC)
	account = links.AccountID(uuid.MustParse("01a19416-ad77-72b3-8c34-8bbbaec97d8b"))
)

func named(t *testing.T, store *links.InMemoryStore) {
	t.Helper()
	link := links.NewLink(account, "4242", "ie", now)
	link.Rename("BraveFox42")
	require.NoError(t, store.SaveLinks(t.Context(), link))
}

func TestADeletedAccountLeavesNoLink(t *testing.T) {
	store, members := links.NewInMemoryStore(), &links.FakeRenamer{}
	require.NoError(t, store.SaveLinks(t.Context(), links.NewLink(account, "4242", "ie", now)))
	useCase := forget_link_usecase.New(store, members)

	require.NoError(t, useCase.Execute(t.Context(), account))
	require.NoError(t, useCase.Execute(t.Context(), account), "a second delivery finds nothing")

	assert.Empty(t, store.Links())
	assert.Empty(t, members.Renamed(), "the bot gave it no nickname")
}

func TestADeletedAccountsUserLosesTheNicknameTheBotGave(t *testing.T) {
	store, members := links.NewInMemoryStore(), &links.FakeRenamer{}
	named(t, store)

	require.NoError(t, forget_link_usecase.New(store, members).Execute(t.Context(), account))

	assert.Equal(t, []string{"4242 "}, members.Renamed())
	assert.Empty(t, store.Links())
}

func TestAUserGoneOrOutOfTheBotsReachIsForgottenAllTheSame(t *testing.T) {
	for name, members := range map[string]*links.FakeRenamer{
		"left the server": {Gone: map[discord.UserID]bool{"4242": true}},
		"above the bot":   {Refused: map[discord.UserID]bool{"4242": true}},
	} {
		t.Run(name, func(t *testing.T) {
			store := links.NewInMemoryStore()
			named(t, store)

			require.NoError(t, forget_link_usecase.New(store, members).Execute(t.Context(), account))

			assert.Empty(t, store.Links())
		})
	}
}

func TestDiscordDownKeepsTheLinkToTryAgain(t *testing.T) {
	store := links.NewInMemoryStore()
	named(t, store)
	down := errors.New("discord answered 502")

	err := forget_link_usecase.New(store, &links.FakeRenamer{Err: down}).Execute(t.Context(), account)

	require.ErrorIs(t, err, down)
	assert.Len(t, store.Links(), 1)
}
