package hear_take_usecase_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/discord"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/nations"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/pings"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/pings/usecases/hear_take_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/roles"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var now = time.Date(2026, 11, 2, 18, 0, 0, 0, time.UTC)

type fixture struct {
	clock   *cptime.FixedClock
	store   *pings.InMemoryStore
	poster  *pings.RecordedPoster
	book    *roles.Book
	useCase *hear_take_usecase.UseCase
}

func setUp() fixture {
	f := fixture{
		clock: cptime.NewFixedClock(now), store: pings.NewInMemoryStore(), poster: &pings.RecordedPoster{},
		book: roles.NewBook(roles.NewInMemoryStore(), &roles.FakeMaker{}),
	}
	herald := pings.NewHerald(f.book, f.poster, pings.Channels{Alerts: "100", Battles: "200"}, "https://clickplanet.lol/play")
	f.useCase = hear_take_usecase.New(pings.NewWatch(), f.clock, f.store, herald, cpatomic.FakeTransactioner{})
	return f
}

func (f fixture) attack(t *testing.T, victim nations.Flag, at time.Time) {
	t.Helper()
	for range pings.AttackTiles {
		require.NoError(t, f.useCase.Execute(t.Context(), pings.Take{Victim: victim, Attacker: "gb", Ground: "us", At: at}))
	}
}

func TestAnAttackOnALinkedFlagPingsItsRole(t *testing.T) {
	f := setUp()
	role, err := f.book.Ensure(t.Context(), roles.Nation("ie"))
	require.NoError(t, err)

	f.attack(t, "ie", now)

	require.Len(t, f.poster.Posts, 1)
	assert.Equal(t, discord.ChannelID("100"), f.poster.Posts[0].Channel)
	assert.Equal(t, role, f.poster.Posts[0].Mention)
	history, err := f.store.AttackPingsOf(t.Context(), "ie", now.Add(-pings.Day))
	require.NoError(t, err)
	assert.Equal(t, pings.History{now}, history)
}

func TestAFlagNobodyLinkedIsNotPinged(t *testing.T) {
	f := setUp()

	f.attack(t, "ie", now)

	assert.Empty(t, f.poster.Posts)
	history, err := f.store.AttackPingsOf(t.Context(), "ie", now.Add(-pings.Day))
	require.NoError(t, err)
	assert.Empty(t, history)
}

func TestASecondAttackWithinTheQuietTimeIsNotPinged(t *testing.T) {
	f := setUp()
	_, err := f.book.Ensure(t.Context(), roles.Nation("ie"))
	require.NoError(t, err)

	f.attack(t, "ie", now)
	f.clock.Advance(time.Hour)
	f.attack(t, "ie", now.Add(time.Hour))
	f.clock.Advance(pings.QuietFor)
	f.attack(t, "ie", now.Add(time.Hour+pings.QuietFor))

	assert.Len(t, f.poster.Posts, 2)
}

func TestAnOldAttackFromABacklogIsNotPinged(t *testing.T) {
	f := setUp()
	_, err := f.book.Ensure(t.Context(), roles.Nation("ie"))
	require.NoError(t, err)

	f.attack(t, "ie", now.Add(-pings.Fresh-time.Second))

	assert.Empty(t, f.poster.Posts)
}

func TestAPostDiscordRefusedKeepsNoPing(t *testing.T) {
	f := setUp()
	_, err := f.book.Ensure(t.Context(), roles.Nation("ie"))
	require.NoError(t, err)
	f.poster.Err = discord.ErrRefused

	for i := range pings.AttackTiles {
		err = f.useCase.Execute(t.Context(), pings.Take{Victim: "ie", Attacker: "gb", Ground: "us", At: now})
		if i < pings.AttackTiles-1 {
			require.NoError(t, err)
		}
	}

	require.ErrorIs(t, err, discord.ErrRefused)
	history, err := f.store.AttackPingsOf(t.Context(), "ie", now.Add(-pings.Day))
	require.NoError(t, err)
	assert.Empty(t, history)
}
