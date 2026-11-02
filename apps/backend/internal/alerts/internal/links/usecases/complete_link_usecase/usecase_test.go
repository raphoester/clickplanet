package complete_link_usecase_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/discord"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/links"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/links/usecases/complete_link_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/links/usecases/start_link_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/nations"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/roles"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var (
	now     = time.Date(2026, 11, 2, 18, 0, 0, 0, time.UTC)
	account = links.AccountID(uuid.MustParse("01a19416-ad77-72b3-8c34-8bbbaec97d8b"))
	other   = links.AccountID(uuid.MustParse("01a193de-df90-7637-9d85-7256584024a6"))
)

type joined struct {
	User  discord.UserID
	Roles []discord.RoleID
}

type fakeDiscord struct {
	traded []string
	joined []joined
	err    error
}

func (d *fakeDiscord) AuthorizationURL(state string, challenge string) string {
	return "https://discord.test/authorize?state=" + state + "&challenge=" + challenge
}

func (d *fakeDiscord) Exchange(_ context.Context, code string, verifier string) (discord.AccessToken, error) {
	if d.err != nil {
		return "", d.err
	}
	d.traded = append(d.traded, code+"/"+verifier)
	return "the-token", nil
}

func (d *fakeDiscord) UserOf(context.Context, discord.AccessToken) (discord.UserID, error) {
	return "4242", nil
}

func (d *fakeDiscord) Join(_ context.Context, user discord.UserID, _ discord.AccessToken, granted []discord.RoleID) error {
	d.joined = append(d.joined, joined{User: user, Roles: granted})
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
	clock   *cptime.FixedClock
	start   *start_link_usecase.UseCase
	useCase *complete_link_usecase.UseCase
}

func setUp(flag nations.Flag) fixture {
	f := fixture{store: links.NewInMemoryStore(), discord: &fakeDiscord{}, maker: &roles.FakeMaker{}, clock: cptime.NewFixedClock(now)}
	f.start = start_link_usecase.New(&links.SequentialSecrets{}, f.clock, f.store, f.discord)
	f.useCase = complete_link_usecase.New(f.clock, f.store, f.discord, pledged(flag),
		roles.NewBook(roles.NewInMemoryStore(), f.maker))
	return f
}

func TestALinkJoinsThePlayerWithTheRolesOfBattlesAndItsFlag(t *testing.T) {
	f := setUp("ie")
	url, err := f.start.Execute(t.Context(), account)
	require.NoError(t, err)
	assert.Contains(t, url, "state=secret-1")

	require.NoError(t, f.useCase.Execute(t.Context(), complete_link_usecase.In{Account: account, Code: "the-code", State: "secret-1"}))

	assert.Equal(t, []string{"the-code/secret-2"}, f.discord.traded)
	assert.Equal(t, []joined{{User: "4242", Roles: []discord.RoleID{"1", "2"}}}, f.discord.joined)
	assert.Equal(t, []string{"Battles", "Ireland"}, f.maker.Made)
	assert.Equal(t, []links.LinkSnapshot{{Account: account, User: "4242", Flag: "ie", LinkedAt: now}}, f.store.Links())
}

func TestAPlayerWithNoFlagYetGetsTheBattlesRoleAlone(t *testing.T) {
	f := setUp("")
	_, err := f.start.Execute(t.Context(), account)
	require.NoError(t, err)

	require.NoError(t, f.useCase.Execute(t.Context(), complete_link_usecase.In{Account: account, Code: "the-code", State: "secret-1"}))

	assert.Equal(t, []joined{{User: "4242", Roles: []discord.RoleID{"1"}}}, f.discord.joined)
	assert.Equal(t, []links.LinkSnapshot{{Account: account, User: "4242", LinkedAt: now}}, f.store.Links())
}

func TestAFlowIsGoodOnceAndForItsOwnStateOnly(t *testing.T) {
	f := setUp("ie")
	err := f.useCase.Execute(t.Context(), complete_link_usecase.In{Account: account, Code: "the-code", State: "secret-1"})
	require.ErrorIs(t, err, links.ErrFlowInvalid, "nothing started")

	_, err = f.start.Execute(t.Context(), account)
	require.NoError(t, err)
	err = f.useCase.Execute(t.Context(), complete_link_usecase.In{Account: account, Code: "the-code", State: "secret-2"})
	require.ErrorIs(t, err, links.ErrFlowInvalid, "another state")

	_, err = f.start.Execute(t.Context(), account)
	require.NoError(t, err)
	require.NoError(t, f.useCase.Execute(t.Context(), complete_link_usecase.In{Account: account, Code: "the-code", State: "secret-3"}))
	err = f.useCase.Execute(t.Context(), complete_link_usecase.In{Account: account, Code: "the-code", State: "secret-3"})
	assert.ErrorIs(t, err, links.ErrFlowInvalid, "spent")
}

func TestARefusedCodeLinksNothingAndSpendsTheFlow(t *testing.T) {
	f := setUp("ie")
	f.discord.err = discord.ErrRefused
	_, err := f.start.Execute(t.Context(), account)
	require.NoError(t, err)

	err = f.useCase.Execute(t.Context(), complete_link_usecase.In{Account: account, Code: "the-code", State: "secret-1"})
	require.ErrorIs(t, err, discord.ErrRefused)

	assert.Empty(t, f.store.Links())
	_, err = f.store.FlowOf(t.Context(), account)
	assert.ErrorIs(t, err, links.ErrNoFlow)
}

func TestAUserLinkedFromAnotherAccountMovesHere(t *testing.T) {
	f := setUp("ie")
	require.NoError(t, f.store.SaveLinks(t.Context(), links.NewLink(other, "4242", "gb", now.Add(-time.Hour))))
	_, err := f.start.Execute(t.Context(), account)
	require.NoError(t, err)

	require.NoError(t, f.useCase.Execute(t.Context(), complete_link_usecase.In{Account: account, Code: "the-code", State: "secret-1"}))

	assert.Equal(t, []links.LinkSnapshot{{Account: account, User: "4242", Flag: "ie", LinkedAt: now}}, f.store.Links())
}
