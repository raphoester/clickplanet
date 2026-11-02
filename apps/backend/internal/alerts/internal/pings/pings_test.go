package pings_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/nations"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/pings"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/roles"
)

var start = time.Date(2026, 11, 2, 18, 0, 0, 0, time.UTC)

func take(attacker, ground nations.Flag, at time.Duration) pings.Take {
	return pings.Take{Victim: "ie", Attacker: attacker, Ground: ground, At: start.Add(at)}
}

func TestAnAttackIsOneFlagTakingEnoughOfAnotherWithinTheWindow(t *testing.T) {
	watch := pings.NewWatch()
	for i := range pings.AttackTiles - 1 {
		ground := nations.Flag("us")
		if i%3 == 0 {
			ground = "ca"
		}
		_, attacked := watch.Hear(take("gb", ground, time.Duration(i)*time.Second))
		require.False(t, attacked, i)
		_, attacked = watch.Hear(take("fr", "us", time.Duration(i)*time.Second))
		require.False(t, attacked, i)
	}

	attack, attacked := watch.Hear(take("gb", "us", 4*time.Minute))
	require.True(t, attacked)
	assert.Equal(t, pings.Attack{
		Victim: "ie", Attacker: "gb", Ground: "us", Tiles: pings.AttackTiles, At: start.Add(4 * time.Minute),
	}, attack)

	_, attacked = watch.Hear(take("gb", "us", 4*time.Minute))
	assert.False(t, attacked, "the attacker starts again from nothing")
	_, attacked = watch.Hear(take("fr", "us", 4*time.Minute))
	assert.True(t, attacked, "another attacker's takes are kept")
}

func TestTakesOlderThanTheWindowCountNothing(t *testing.T) {
	watch := pings.NewWatch()
	for i := range pings.AttackTiles {
		at := time.Duration(0)
		if i >= pings.AttackTiles/2 {
			at = pings.AttackWindow
		}
		_, attacked := watch.Hear(take("gb", "us", at))
		require.False(t, attacked, i)
	}
}

func TestAFlagIsPingedAtMostOnceInThreeHoursAndTwiceADay(t *testing.T) {
	assert.True(t, pings.History{}.Allows(start))
	assert.False(t, pings.History{start.Add(-pings.QuietFor + time.Minute)}.Allows(start))
	assert.True(t, pings.History{start.Add(-pings.QuietFor)}.Allows(start))
	assert.False(t, pings.History{start.Add(-10 * time.Hour), start.Add(-5 * time.Hour)}.Allows(start))
	assert.True(t, pings.History{start.Add(-pings.Day), start.Add(-5 * time.Hour)}.Allows(start))
}

func TestAnAttackOrAStartPastItsMinutesIsStale(t *testing.T) {
	assert.False(t, pings.Attack{At: start}.Stale(start.Add(pings.Fresh)))
	assert.True(t, pings.Attack{At: start}.Stale(start.Add(pings.Fresh+time.Second)))
	assert.False(t, pings.Start{StartsAt: start}.Stale(start.Add(pings.Fresh)))
	assert.True(t, pings.Start{StartsAt: start}.Stale(start.Add(pings.Fresh+time.Second)))
}

func herald(poster *pings.RecordedPoster) (pings.Herald, *roles.Book) {
	book := roles.NewBook(roles.NewInMemoryStore(), &roles.FakeMaker{})
	return pings.NewHerald(book, poster, pings.Channels{Alerts: "100", Battles: "200"}, "https://clickplanet.lol/play"), book
}

func TestTheHeraldCallsTheVictimsRoleToDefend(t *testing.T) {
	poster := &pings.RecordedPoster{}
	herald, book := herald(poster)

	err := herald.CallToDefend(t.Context(), pings.Attack{Victim: "ie", Attacker: "gb", Ground: "us", Tiles: 240})
	require.ErrorIs(t, err, roles.ErrNoRole)
	assert.Empty(t, poster.Posts)

	role, err := book.Ensure(t.Context(), roles.Nation("ie"))
	require.NoError(t, err)
	require.NoError(t, herald.CallToDefend(t.Context(), pings.Attack{Victim: "ie", Attacker: "gb", Ground: "us", Tiles: 240}))

	assert.Equal(t, []pings.Post{{
		Channel: "100",
		Text:    "<@&1> 🇬🇧 United Kingdom took 240 of your tiles in 🇺🇸 United States in 5 minutes. https://clickplanet.lol/play",
		Mention: role,
	}}, poster.Posts)
}

func TestTheHeraldCallsEveryoneToBattle(t *testing.T) {
	poster := &pings.RecordedPoster{}
	herald, _ := herald(poster)
	ends := time.Unix(1793570400, 0)

	for _, started := range []pings.Start{
		{Kind: pings.BattleStart, Times: 2, EndsAt: ends},
		{Kind: pings.FinaleStart, Season: 1, Times: 3, EndsAt: ends},
		{Kind: pings.SeasonStart, Season: 2, EndsAt: ends},
	} {
		require.NoError(t, herald.CallToBattle(t.Context(), started))
	}
	require.ErrorIs(t, herald.CallToBattle(t.Context(), pings.Start{}), pings.ErrUnknownStart)

	assert.Equal(t, []pings.Post{
		{Channel: "200", Mention: "1", Text: "<@&1> ⚔️ The battle is on: 2x points until <t:1793570400:t>. https://clickplanet.lol/play"},
		{Channel: "200", Mention: "1", Text: "<@&1> ⚔️ The Final Battle of season 1 is on: 3x points until <t:1793570400:t>. https://clickplanet.lol/play"},
		{Channel: "200", Mention: "1", Text: "<@&1> 🌍 Season 2 has started on a blank map. https://clickplanet.lol/play"},
	}, poster.Posts)
}
