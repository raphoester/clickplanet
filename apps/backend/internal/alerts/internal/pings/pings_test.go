package pings_test

import (
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/discord"
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

func pointAt(lat, lon float64) pings.Point {
	lat, lon = lat*math.Pi/180, lon*math.Pi/180
	return pings.Point{X: -math.Cos(lat) * math.Cos(lon), Y: math.Sin(lat), Z: math.Cos(lat) * math.Sin(lon)}
}

func placed(attacker, ground nations.Flag, point pings.Point) pings.Take {
	placed := take(attacker, ground, time.Minute)
	placed.Point = point
	return placed
}

// Even over a disc of the radius, as a sunflower's seeds.
func discAround(lat, lon, radius float64, count int) []pings.Point {
	golden := math.Pi * (3 - math.Sqrt(5))
	points := make([]pings.Point, 0, count)
	for i := range count {
		r := radius * math.Sqrt((float64(i)+0.5)/float64(count))
		angle := float64(i) * golden
		points = append(points, pointAt(lat+r*math.Cos(angle), lon+r*math.Sin(angle)/math.Cos(lat*math.Pi/180)))
	}
	return points
}

func TestAnAttacksViewFramesTheAttackersTakesOnItsGround(t *testing.T) {
	watch := pings.NewWatch()
	_, attacked := watch.Hear(placed("fr", "us", pointAt(90, 0)))
	require.False(t, attacked)
	_, attacked = watch.Hear(placed("gb", "ca", pointAt(90, 0)))
	require.False(t, attacked)
	disc := discAround(40, -100, 5, pings.AttackTiles-1)
	for i, point := range disc[:len(disc)-1] {
		_, attacked = watch.Hear(placed("gb", "us", point))
		require.False(t, attacked, i)
	}

	attack, attacked := watch.Hear(placed("gb", "us", disc[len(disc)-1]))
	require.True(t, attacked)
	assert.InDelta(t, 40, attack.View.Lat, 0.2)
	assert.InDelta(t, -100, attack.View.Lon, 0.2)
	nearest := math.Sqrt(0.9) * 5 * math.Pi / 180
	assert.InDelta(t, 0.4/nearest, attack.View.Zoom, 0.1, "the nearest 90%% of the takes fill 40%% of the screen's half height")
}

func TestAFewStrayTakesDoNotZoomTheViewOut(t *testing.T) {
	watch := pings.NewWatch()
	for i := range pings.AttackTiles - 1 {
		point := pointAt(-25, 134)
		if i%20 == 0 {
			point = pointAt(10, 100)
		}
		_, attacked := watch.Hear(placed("gb", "au", point))
		require.False(t, attacked, i)
	}

	attack, attacked := watch.Hear(placed("gb", "au", pointAt(-25, 134)))
	require.True(t, attacked)
	assert.InDelta(t, 20, attack.View.Zoom, 1e-9, "as close as a view comes")
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
		Channel:  "100",
		Text:     "<@&1> 🇬🇧 United Kingdom took 240 of your tiles in 🇺🇸 United States in 5 minutes. https://clickplanet.lol/play",
		Audience: discord.RoleAudience(role),
	}}, poster.Posts)
}

func TestTheHeraldLinksToTheAttacksView(t *testing.T) {
	poster := &pings.RecordedPoster{}
	herald, book := herald(poster)
	_, err := book.Ensure(t.Context(), roles.Nation("ie"))
	require.NoError(t, err)

	require.NoError(t, herald.CallToDefend(t.Context(), pings.Attack{
		Victim: "ie", Attacker: "gb", Ground: "us", Tiles: 240, View: pings.View{Lat: 40.4167, Lon: -100.0049, Zoom: 2.9849},
	}))

	require.Len(t, poster.Posts, 1)
	assert.Equal(t, "<@&1> 🇬🇧 United Kingdom took 240 of your tiles in 🇺🇸 United States in 5 minutes. "+
		"https://clickplanet.lol/play?lat=40.42&lon=-100.00&zoom=3.0", poster.Posts[0].Text)
}

func TestTheHeraldCallsEverybodyInTheServerToBattle(t *testing.T) {
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
		{Channel: "200", Audience: discord.Everybody(), Text: "@everyone ⚔️ The battle is on: 2x points until <t:1793570400:t>. https://clickplanet.lol/play"},
		{Channel: "200", Audience: discord.Everybody(), Text: "@everyone ⚔️ The Final Battle of season 1 is on: 3x points until <t:1793570400:t>. https://clickplanet.lol/play"},
		{Channel: "200", Audience: discord.Everybody(), Text: "@everyone 🌍 Season 2 has started on a blank map. https://clickplanet.lol/play"},
	}, poster.Posts)
}

func TestTheHeraldTellsTheDaysRankingAndPingsItsTopThree(t *testing.T) {
	poster := &pings.RecordedPoster{}
	herald, book := herald(poster)
	ireland, err := book.Ensure(t.Context(), roles.Nation("ie"))
	require.NoError(t, err)
	france, err := book.Ensure(t.Context(), roles.Nation("fr"))
	require.NoError(t, err)
	_, err = book.Ensure(t.Context(), roles.Nation("es"))
	require.NoError(t, err)

	won := []pings.Place{
		{Flag: "ie", Rank: 1, Points: 25},
		{Flag: "fr", Rank: 2, Points: 18},
		{Flag: "gb", Rank: 2, Points: 18},
		{Flag: "es", Rank: 4, Points: 12},
		{Flag: "de", Rank: 5, Points: 10},
		{Flag: "it", Rank: 6, Points: 8},
		{Flag: "pt", Rank: 7, Points: 6},
		{Flag: "be", Rank: 8, Points: 4},
		{Flag: "nl", Rank: 9, Points: 2},
		{Flag: "xb", Rank: 10, Points: 1},
	}
	table := []pings.Place{
		{Flag: "fr", Rank: 1, Points: 118},
		{Flag: "ie", Rank: 2, Points: 97},
		{Flag: "gb", Rank: 3, Points: 60},
		{Flag: "es", Rank: 4, Points: 40},
		{Flag: "de", Rank: 5, Points: 22},
		{Flag: "it", Rank: 6, Points: 20},
		{Flag: "pt", Rank: 7, Points: 12},
		{Flag: "be", Rank: 8, Points: 9},
		{Flag: "nl", Rank: 9, Points: 2},
		{Flag: "xb", Rank: 10, Points: 1},
		{Flag: "pl", Rank: 10, Points: 1},
		{Flag: "at", Rank: 12, Points: 1},
	}
	require.NoError(t, herald.TellRanking(t.Context(),
		pings.EndedDay{Round: "season-1/day-3", Season: 1, Number: 3}, pings.Ranking{Won: won, Table: table}))

	assert.Equal(t, []pings.Post{{
		Channel:  "100",
		Audience: discord.RoleAudience(ireland, france),
		Text: "🏁 **Day 3 of season 1 is over.** The points it gave:\n" +
			"🥇 🇮🇪 <@&1> +25\n" +
			"🥈 🇫🇷 <@&2> +18\n" +
			"🥈 🇬🇧 United Kingdom +18\n" +
			"4️⃣ 🇪🇸 Spain +12\n" +
			"5️⃣ 🇩🇪 Germany +10\n" +
			"6️⃣ 🇮🇹 Italy +8\n" +
			"7️⃣ 🇵🇹 Portugal +6\n" +
			"8️⃣ 🇧🇪 Belgium +4\n" +
			"9️⃣ 🇳🇱 Netherlands +2\n" +
			"🔟 🏴 Brittany +1\n" +
			"🏆 **The season's ranking:**\n" +
			"🥇 🇫🇷 France 118 points\n" +
			"🥈 🇮🇪 Ireland 97 points\n" +
			"🥉 🇬🇧 United Kingdom 60 points\n" +
			"4️⃣ 🇪🇸 Spain 40 points\n" +
			"5️⃣ 🇩🇪 Germany 22 points\n" +
			"6️⃣ 🇮🇹 Italy 20 points\n" +
			"7️⃣ 🇵🇹 Portugal 12 points\n" +
			"8️⃣ 🇧🇪 Belgium 9 points\n" +
			"9️⃣ 🇳🇱 Netherlands 2 points\n" +
			"🔟 🏴 Brittany 1 point\n" +
			"🔟 🇵🇱 Poland 1 point\n" +
			"https://clickplanet.lol/play",
	}}, poster.Posts)
}

func TestARankingWhoseTopThreeNobodyLinkedPingsNobody(t *testing.T) {
	poster := &pings.RecordedPoster{}
	herald, _ := herald(poster)

	require.NoError(t, herald.TellRanking(t.Context(), pings.EndedDay{Round: "season-1/day-3", Season: 1, Number: 3},
		pings.Ranking{Won: []pings.Place{{Flag: "ie", Rank: 1, Points: 25}}, Table: []pings.Place{{Flag: "ie", Rank: 1, Points: 25}}}))

	require.Len(t, poster.Posts, 1)
	assert.Equal(t, discord.RoleAudience(), poster.Posts[0].Audience)
	assert.False(t, poster.Posts[0].Audience.Everybody())
	assert.NotContains(t, poster.Posts[0].Text, "@everyone")
}

func TestADayIsStaleAnHourAfterItsEnd(t *testing.T) {
	assert.False(t, pings.EndedDay{EndedAt: start}.Stale(start.Add(pings.RankingFresh)))
	assert.True(t, pings.EndedDay{EndedAt: start}.Stale(start.Add(pings.RankingFresh+time.Second)))
}
