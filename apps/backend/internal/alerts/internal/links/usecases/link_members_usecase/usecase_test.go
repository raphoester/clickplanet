package link_members_usecase_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/discord"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/links"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/links/usecases/link_members_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/nations"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/roles"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var (
	now = time.Date(2026, 11, 2, 18, 0, 0, 0, time.UTC)
	ada = links.AccountID{15: 1}
	bob = links.AccountID{15: 2}
)

// auth's accounts signed in with Discord, a page at a time in the order of their ids, as GetDiscordUsers answers.
type fakeAccounts struct {
	signed []links.DiscordAccount
	err    error
	asked  int
}

func (a *fakeAccounts) DiscordAccounts(_ context.Context, after links.AccountID, limit int) ([]links.DiscordAccount, error) {
	a.asked++
	if a.err != nil {
		return nil, a.err
	}
	sorted := slices.Clone(a.signed)
	slices.SortFunc(sorted, func(x, y links.DiscordAccount) int { return bytes.Compare(x.Account[:], y.Account[:]) })
	var page []links.DiscordAccount
	for _, signed := range sorted {
		if bytes.Compare(signed.Account[:], after[:]) > 0 && len(page) < limit {
			page = append(page, signed)
		}
	}
	return page, nil
}

// The server's members, and the roles granted. A user in failing makes Discord fail for it; one in leaving left the
// server between the check and the grant.
type fakeMembers struct {
	in      *cpcolls.Set[discord.UserID]
	failing *cpcolls.Set[discord.UserID]
	leaving *cpcolls.Set[discord.UserID]
	checked []discord.UserID
	granted []string
}

func (m *fakeMembers) IsMember(_ context.Context, user discord.UserID) (bool, error) {
	m.checked = append(m.checked, user)
	if m.failing.Contains(user) {
		return false, errors.New("discord answered 500")
	}
	return m.in.Contains(user), nil
}

func (m *fakeMembers) Grant(_ context.Context, user discord.UserID, role discord.RoleID) error {
	if m.leaving.Contains(user) {
		return discord.ErrNotMember
	}
	m.granted = append(m.granted, string(user)+" "+string(role))
	return nil
}

type pledged map[links.AccountID]nations.Flag

func (p pledged) FlagOf(_ context.Context, account links.AccountID) (nations.Flag, error) {
	return p[account], nil
}

type fixture struct {
	store    *links.InMemoryStore
	accounts *fakeAccounts
	members  *fakeMembers
	maker    *roles.FakeMaker
	flags    pledged
	names    *links.FakeNames
	renamer  *links.FakeRenamer
	useCase  *link_members_usecase.UseCase
}

func setUp() *fixture {
	f := &fixture{
		store:    links.NewInMemoryStore(),
		accounts: &fakeAccounts{},
		members: &fakeMembers{
			in: cpcolls.NewSet[discord.UserID](), failing: cpcolls.NewSet[discord.UserID](), leaving: cpcolls.NewSet[discord.UserID](),
		},
		maker:   &roles.FakeMaker{},
		flags:   pledged{},
		names:   links.NewFakeNames(),
		renamer: &links.FakeRenamer{},
	}
	f.useCase = link_members_usecase.New(cptime.NewFixedClock(now), f.store, f.accounts, f.members, f.flags,
		roles.NewBook(roles.NewInMemoryStore(), f.maker), links.NewNamer(f.store, f.names, f.renamer))
	return f
}

func (f *fixture) signedIn(account links.AccountID, user discord.UserID, flag nations.Flag, member bool) {
	f.accounts.signed = append(f.accounts.signed, links.DiscordAccount{Account: account, User: user})
	f.flags[account] = flag
	if member {
		f.members.in.Add(user)
	}
}

func TestAMemberAlreadyInTheServerIsLinkedWithItsFlagsRole(t *testing.T) {
	f := setUp()
	f.signedIn(ada, "1111", "ie", true)

	out, err := f.useCase.Execute(t.Context())
	require.NoError(t, err)

	assert.Equal(t, link_members_usecase.Out{Linked: 1}, out)
	assert.Equal(t, []string{"Ireland"}, f.maker.Made)
	assert.Equal(t, []string{"1111 1"}, f.members.granted)
	assert.Equal(t, []links.LinkSnapshot{{Account: ada, User: "1111", Flag: "ie", LinkedAt: now}}, f.store.Links())
}

func TestAMemberWithNoFlagYetIsLinkedWithNoRole(t *testing.T) {
	f := setUp()
	f.signedIn(ada, "1111", "", true)

	out, err := f.useCase.Execute(t.Context())
	require.NoError(t, err)

	assert.Equal(t, link_members_usecase.Out{Linked: 1}, out)
	assert.Empty(t, f.members.granted)
	assert.Equal(t, []links.LinkSnapshot{{Account: ada, User: "1111", LinkedAt: now}}, f.store.Links())
}

func TestAUserNotInTheServerIsLeftOutside(t *testing.T) {
	f := setUp()
	f.signedIn(ada, "1111", "ie", false)

	out, err := f.useCase.Execute(t.Context())
	require.NoError(t, err)

	assert.Equal(t, link_members_usecase.Out{Outside: 1}, out)
	assert.Empty(t, f.members.granted)
	assert.Empty(t, f.store.Links())
}

func TestAMemberWhoLeftBeforeTheGrantIsOutside(t *testing.T) {
	f := setUp()
	f.signedIn(ada, "1111", "ie", true)
	f.members.leaving.Add("1111")

	out, err := f.useCase.Execute(t.Context())
	require.NoError(t, err)

	assert.Equal(t, link_members_usecase.Out{Outside: 1}, out)
	assert.Empty(t, f.store.Links())
}

func TestALinkedAccountIsLeftAsItIs(t *testing.T) {
	f := setUp()
	f.signedIn(ada, "1111", "ie", true)
	require.NoError(t, f.store.SaveLinks(t.Context(), links.NewLink(ada, "1111", "gb", now.Add(-time.Hour))))

	out, err := f.useCase.Execute(t.Context())
	require.NoError(t, err)

	assert.Equal(t, link_members_usecase.Out{}, out)
	assert.Empty(t, f.members.checked, "Discord is not asked")
	assert.Equal(t, []links.LinkSnapshot{{Account: ada, User: "1111", Flag: "gb", LinkedAt: now.Add(-time.Hour)}},
		f.store.Links())
}

func TestAUserLinkedFromAnotherAccountIsNotMoved(t *testing.T) {
	f := setUp()
	f.signedIn(ada, "1111", "ie", true)
	require.NoError(t, f.store.SaveLinks(t.Context(), links.NewLink(bob, "1111", "gb", now.Add(-time.Hour))))

	out, err := f.useCase.Execute(t.Context())
	require.NoError(t, err)

	assert.Equal(t, link_members_usecase.Out{}, out)
	assert.Equal(t, []links.LinkSnapshot{{Account: bob, User: "1111", Flag: "gb", LinkedAt: now.Add(-time.Hour)}},
		f.store.Links())
}

func TestAnAccountThatFailsIsCountedAndTheOthersGoOn(t *testing.T) {
	f := setUp()
	f.signedIn(ada, "1111", "ie", true)
	f.signedIn(bob, "2222", "fr", true)
	f.members.failing.Add("1111")

	out, err := f.useCase.Execute(t.Context())
	require.NoError(t, err)

	assert.Equal(t, 1, out.Linked)
	require.Len(t, out.Failed, 1)
	assert.Equal(t, ada, out.Failed[0].Account)
	assert.Equal(t, discord.UserID("1111"), out.Failed[0].User)
	assert.Equal(t, []links.LinkSnapshot{{Account: bob, User: "2222", Flag: "fr", LinkedAt: now}}, f.store.Links())
}

func TestASecondRunLinksNobodyTwice(t *testing.T) {
	f := setUp()
	f.signedIn(ada, "1111", "ie", true)
	f.signedIn(bob, "2222", "fr", false)

	first, err := f.useCase.Execute(t.Context())
	require.NoError(t, err)
	second, err := f.useCase.Execute(t.Context())
	require.NoError(t, err)

	assert.Equal(t, link_members_usecase.Out{Linked: 1, Outside: 1}, first)
	assert.Equal(t, link_members_usecase.Out{Outside: 1}, second, "the one outside is asked again: it may have joined since")
	assert.Equal(t, []string{"1111 1"}, f.members.granted)
}

func TestEveryPageIsRead(t *testing.T) {
	f := setUp()
	for i := range link_members_usecase.PageSize + 1 {
		account := links.AccountID{14: byte((i + 1) >> 8), 15: byte(i + 1)}
		f.signedIn(account, discord.UserID(fmt.Sprint(1000+i)), "", true)
	}

	out, err := f.useCase.Execute(t.Context())
	require.NoError(t, err)

	assert.Equal(t, link_members_usecase.PageSize+1, out.Linked)
	assert.Equal(t, 2, f.accounts.asked)
	assert.Len(t, f.store.Links(), link_members_usecase.PageSize+1)
}

func TestAMemberLinkedNowIsGivenItsUsername(t *testing.T) {
	f := setUp()
	f.signedIn(ada, "1111", "ie", true)
	f.names.Name(ada, "BraveFox42")

	out, err := f.useCase.Execute(t.Context())
	require.NoError(t, err)

	assert.Equal(t, link_members_usecase.Out{Linked: 1, Named: 1}, out)
	assert.Equal(t, []string{"1111 BraveFox42"}, f.renamer.Renamed())
	assert.Equal(t, []links.LinkSnapshot{{Account: ada, User: "1111", Flag: "ie", Name: "BraveFox42", LinkedAt: now}},
		f.store.Links())
}

func TestAnAccountLinkedBeforeIsGivenItsUsernameOnce(t *testing.T) {
	f := setUp()
	f.signedIn(ada, "1111", "ie", true)
	require.NoError(t, f.store.SaveLinks(t.Context(), links.NewLink(ada, "1111", "gb", now.Add(-time.Hour))))
	f.names.Name(ada, "BraveFox42")

	first, err := f.useCase.Execute(t.Context())
	require.NoError(t, err)
	second, err := f.useCase.Execute(t.Context())
	require.NoError(t, err)

	assert.Equal(t, link_members_usecase.Out{Named: 1}, first)
	assert.Equal(t, link_members_usecase.Out{}, second)
	assert.Empty(t, f.members.checked, "Discord is not asked whether it is in the server")
	assert.Equal(t, []string{"1111 BraveFox42"}, f.renamer.Renamed())
}

func TestAUserOutsideOrLinkedFromAnotherAccountIsNotNamed(t *testing.T) {
	f := setUp()
	f.signedIn(ada, "1111", "ie", false)
	f.signedIn(bob, "2222", "fr", true)
	require.NoError(t, f.store.SaveLinks(t.Context(), links.NewLink(links.AccountID{15: 3}, "2222", "fr", now)))
	f.names.Name(ada, "Ada")
	f.names.Name(bob, "Bob")

	out, err := f.useCase.Execute(t.Context())
	require.NoError(t, err)

	assert.Equal(t, link_members_usecase.Out{Outside: 1}, out)
	assert.Empty(t, f.renamer.Renamed())
}

func TestANicknameDiscordRefusedIsAFailureAndTheLinkKept(t *testing.T) {
	f := setUp()
	f.signedIn(ada, "1111", "ie", true)
	f.names.Name(ada, "TheOwner")
	f.renamer.Refused = map[discord.UserID]bool{"1111": true}

	out, err := f.useCase.Execute(t.Context())
	require.NoError(t, err)

	assert.Equal(t, 1, out.Linked)
	assert.Zero(t, out.Named)
	require.Len(t, out.Failed, 1)
	assert.ErrorIs(t, out.Failed[0].Err, discord.ErrRefused)
	assert.Equal(t, []links.LinkSnapshot{{Account: ada, User: "1111", Flag: "ie", LinkedAt: now}}, f.store.Links(),
		"a run again tries to name it")
}

func TestAuthThatDoesNotAnswerStopsTheRun(t *testing.T) {
	f := setUp()
	f.accounts.err = errors.New("auth does not answer")

	_, err := f.useCase.Execute(t.Context())

	require.Error(t, err)
	assert.Empty(t, f.members.checked)
}

func TestARunCutShortStops(t *testing.T) {
	f := setUp()
	f.signedIn(ada, "1111", "ie", true)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := f.useCase.Execute(ctx)

	require.ErrorIs(t, err, context.Canceled)
	assert.Empty(t, f.members.checked)
}
