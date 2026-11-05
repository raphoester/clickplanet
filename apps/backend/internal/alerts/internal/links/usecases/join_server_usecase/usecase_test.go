package join_server_usecase_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/discord"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/links"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/links/usecases/join_server_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/nations"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/roles"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var (
	now     = time.Date(2026, 11, 2, 18, 0, 0, 0, time.UTC)
	account = links.AccountID(uuid.MustParse("01a19416-ad77-72b3-8c34-8bbbaec97d8b"))
	other   = links.AccountID(uuid.MustParse("01a193de-df90-7637-9d85-7256584024a6"))
	signIn  = join_server_usecase.In{Account: account, User: "4242", Token: "the-token"}
)

type joined struct {
	User  discord.UserID
	Token discord.AccessToken
	Roles []discord.RoleID
}

type fakeDiscord struct {
	joined []joined
	err    error
}

func (d *fakeDiscord) Join(_ context.Context, user discord.UserID, token discord.AccessToken, granted []discord.RoleID) error {
	if d.err != nil {
		return d.err
	}
	d.joined = append(d.joined, joined{User: user, Token: token, Roles: granted})
	return nil
}

type pledged nations.Flag

func (p pledged) FlagOf(context.Context, links.AccountID) (nations.Flag, error) {
	return nations.Flag(p), nil
}

type fixture struct {
	store   *links.InMemoryStore
	discord *fakeDiscord
	maker   *roles.FakeMaker
	useCase *join_server_usecase.UseCase
}

func setUp(flag nations.Flag) fixture {
	f := fixture{store: links.NewInMemoryStore(), discord: &fakeDiscord{}, maker: &roles.FakeMaker{}}
	f.useCase = join_server_usecase.New(cptime.NewFixedClock(now), f.store, f.discord, pledged(flag),
		roles.NewBook(roles.NewInMemoryStore(), f.maker))
	return f
}

func TestASignInJoinsThePlayerWithItsFlagsRole(t *testing.T) {
	f := setUp("ie")

	require.NoError(t, f.useCase.Execute(t.Context(), signIn))

	assert.Equal(t, []joined{{User: "4242", Token: "the-token", Roles: []discord.RoleID{"1"}}}, f.discord.joined)
	assert.Equal(t, []string{"Ireland"}, f.maker.Made)
	assert.Equal(t, []links.LinkSnapshot{{Account: account, User: "4242", Flag: "ie", LinkedAt: now}}, f.store.Links())
}

func TestAPlayerWithNoFlagYetJoinsWithNoRole(t *testing.T) {
	f := setUp("")

	require.NoError(t, f.useCase.Execute(t.Context(), signIn))

	assert.Equal(t, []joined{{User: "4242", Token: "the-token"}}, f.discord.joined)
	assert.Empty(t, f.maker.Made)
	assert.Equal(t, []links.LinkSnapshot{{Account: account, User: "4242", LinkedAt: now}}, f.store.Links())
}

func TestASignInAgainOfALinkedPlayerAddsNobodyBack(t *testing.T) {
	f := setUp("ie")
	require.NoError(t, f.store.SaveLinks(t.Context(), links.NewLink(account, "4242", "gb", now.Add(-time.Hour))))

	require.NoError(t, f.useCase.Execute(t.Context(), signIn))

	assert.Empty(t, f.discord.joined)
	assert.Equal(t, []links.LinkSnapshot{{Account: account, User: "4242", Flag: "gb", LinkedAt: now.Add(-time.Hour)}},
		f.store.Links())
}

func TestARefusedJoinLinksNothing(t *testing.T) {
	f := setUp("ie")
	f.discord.err = discord.ErrRefused

	require.ErrorIs(t, f.useCase.Execute(t.Context(), signIn), discord.ErrRefused)

	assert.Empty(t, f.store.Links())
}

func TestAUserLinkedFromAnotherAccountMovesHere(t *testing.T) {
	f := setUp("ie")
	require.NoError(t, f.store.SaveLinks(t.Context(), links.NewLink(other, "4242", "gb", now.Add(-time.Hour))))

	require.NoError(t, f.useCase.Execute(t.Context(), signIn))

	assert.Equal(t, []links.LinkSnapshot{{Account: account, User: "4242", Flag: "ie", LinkedAt: now}}, f.store.Links())
}
