package posts_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/posts"
)

var (
	foundAt = time.Date(2026, 10, 26, 20, 29, 0, 0, time.UTC)
	day18   = posts.StoryKey{Flag: "pw", Continent: "the world", Angle: posts.DayWon, Round: "0.18"}
)

// The evening of 2026-10-26: France and Turkey at war over Turkey and Syria, France taking India from Algeria.
func findings(t *testing.T) posts.Findings {
	t.Helper()

	found, err := posts.FindingsOf(posts.Findings{
		Since: foundAt.Add(-12 * time.Hour), Until: foundAt,
		Fronts: []posts.Front{{
			Since: foundAt.Add(-5 * time.Hour), Until: foundAt, TilesChanged: 9800, Captured: 9000,
			Countries: []posts.FrontCountry{
				{Code: "tr", Continent: "Middle East", Tiles: 2500, Changed: 6000, HoldersStart: map[string]int{"tr": 2000}, HoldersEnd: map[string]int{"fr": 1500, "tr": 900}, Takers: map[string]int{"fr": 5000, "tr": 200}},
				{Code: "sy", Continent: "Middle East", Tiles: 600, Changed: 400, HoldersStart: map[string]int{"tr": 300}, HoldersEnd: map[string]int{"fr": 300}, Takers: map[string]int{"fr": 300}},
				{Code: "in", Continent: "South Asia", Tiles: 6300, Changed: 3400, HoldersStart: map[string]int{"dz": 2600}, HoldersEnd: map[string]int{"fr": 6100}, Takers: map[string]int{"fr": 3400}},
			},
			Flags: []posts.FrontFlag{
				{Code: "fr", Continent: "Europe", Took: 8700, Lost: 3000, TookFrom: map[string]int{"tr": 5300, "dz": 3400}, Solidity: 0.86},
				{Code: "tr", Continent: "Middle East", Took: 3000, Lost: 5300, TookFrom: map[string]int{"fr": 3000}, Solidity: 0.9},
				{Code: "dz", Continent: "Africa", Lost: 3400},
			},
			Bombs: 3, Score: 3886.5,
		}},
		Tours: []posts.Tour{{
			Flag: "fr", Since: foundAt.Add(-12 * time.Hour), Until: foundAt, Took: 27000, ByContinent: map[string]int{"Europe": 12000, "South Asia": 9000}, NextTook: 9000,
		}},
		Places: []posts.Place{
			{Kind: posts.ContinentPlace, Continent: "South Asia", Tiles: 9600, Holder: "fr", Held: 6170, Share: 0.64, Changed: 6000},
			{Kind: posts.World, Tiles: 262000, Holder: "pw", Held: 74000, Share: 0.28, Changed: 7000},
		},
		Seasonal:   []posts.Seasonal{{Title: "PALAU WINS DAY 18", Key: day18, Countries: []string{"pw", "dz"}}},
		Names:      map[string]string{"fr": "France", "tr": "Turkey", "dz": "Algeria", "in": "India", "sy": "Syria", "pw": "Palau"},
		Continents: map[string]string{"Europe": "eu", "Middle East": "", "South Asia": "", "Africa": ""},
	})
	require.NoError(t, err)
	return found
}

func TestAStoryOfAFrontHoldsWhenItsFlagsAndItsPlaceAreTheFrontsAndItsKeyIsItsFirstSideThere(t *testing.T) {
	found := findings(t)

	battle, err := found.Check(posts.Story{Front: 1, Kind: posts.Battle, Attacker: "fr", Rival: "tr", TakenFrom: []string{"tr"}, Countries: []string{"tr", "sy"}})
	require.NoError(t, err)
	kickout, err := found.Check(posts.Story{Front: 1, Kind: posts.Kickout, Attacker: "tr", TakenFrom: []string{"fr"}, Countries: []string{"tr"}})
	require.NoError(t, err)
	rout, err := found.Check(posts.Story{Front: 1, Kind: posts.Rout, Attacker: "fr", TakenFrom: []string{"dz"}, Countries: []string{"in"}})
	require.NoError(t, err)
	attack, err := found.Check(posts.Story{Front: 1, Kind: posts.Attack, Attacker: "fr", TakenFrom: []string{"tr", "dz"}, Continent: "Middle East"})
	require.NoError(t, err)
	comeback, err := found.Check(posts.Story{Front: 1, Kind: posts.Comeback, Attacker: "tr", TakenFrom: []string{"fr"}, Continent: "Middle East", Team: "Middle East"})
	require.NoError(t, err)

	assert.Equal(t, posts.StoryKey{Flag: "fr", Continent: "Middle East"}, battle)
	assert.Equal(t, posts.StoryKey{Flag: "tr", Continent: "Middle East"}, kickout)
	assert.Equal(t, posts.StoryKey{Flag: "dz", Continent: "South Asia"}, rout, "a rout is the loser's story")
	assert.Equal(t, posts.StoryKey{Flag: "fr", Continent: "Middle East"}, attack)
	assert.Equal(t, posts.StoryKey{Flag: "tr", Continent: "Middle East"}, comeback, "a continent with no flag of its own is its leading flag's")
}

func TestAStoryThatSaysWhatTheFrontsFactsDoNotIsRefusedWithWhy(t *testing.T) {
	found := findings(t)
	fr := posts.Story{Front: 1, Kind: posts.Invasion, Attacker: "fr", TakenFrom: []string{"tr"}, Countries: []string{"tr"}}

	for why, story := range map[string]posts.Story{
		"no front 2 of 1":                    {Front: 2, Kind: posts.Invasion, Attacker: "fr", Countries: []string{"tr"}},
		`"holding" is no kind a front tells`: with(fr, func(s *posts.Story) { s.Kind = "holding" }),
		`"dz" took nothing`:                  with(fr, func(s *posts.Story) { s.Attacker = "dz" }),
		`"pw" took nothing`:                  with(fr, func(s *posts.Story) { s.Attacker = "pw" }),
		`"fr" took from itself`:              with(fr, func(s *posts.Story) { s.TakenFrom = []string{"fr"} }),
		`"pw" lost nothing`:                  with(fr, func(s *posts.Story) { s.TakenFrom = []string{"pw"} }),
		`"fr" and "dz" did not take from each other`: with(fr, func(s *posts.Story) {
			s.Kind = posts.Battle
			s.Rival = "dz"
		}),
		"only a battle has a rival": with(fr, func(s *posts.Story) { s.Rival = "tr" }),
		"a rout names the flag thrown out": with(fr, func(s *posts.Story) {
			s.Kind = posts.Rout
			s.TakenFrom = nil
		}),
		"one or two countries, or on a continent": with(fr, func(s *posts.Story) { s.Countries = nil }),
		"told in one or two countries, not 3":     with(fr, func(s *posts.Story) { s.Countries = []string{"tr", "sy", "in"} }),
		`"de" is no country of it`:                with(fr, func(s *posts.Story) { s.Countries = []string{"de"} }),
		`"Europe" is no continent of it`: with(fr, func(s *posts.Story) {
			s.Countries = nil
			s.Continent = "Europe"
		}),
		`"tr" took nothing in [in]`:                                               with(fr, func(s *posts.Story) { s.Attacker = "tr"; s.TakenFrom = []string{"fr"}; s.Countries = []string{"in"} }),
		`"fr" is not at home in [tr], so it is no comeback`:                       with(fr, func(s *posts.Story) { s.Kind = posts.Comeback }),
		`"fr" held nothing in [tr] when the front opened, so it kicks nobody out`: with(fr, func(s *posts.Story) { s.Kind = posts.Kickout }),
		`"tr" held nothing in [in] when the front opened, so it is not thrown out`: with(fr, func(s *posts.Story) {
			s.Kind = posts.Rout
			s.Countries = []string{"in"}
			s.TakenFrom = []string{"tr", "dz"}
		}),
		`a team is a continent's flags striking back together on it, and "Middle East" is not that`: with(fr, func(s *posts.Story) { s.Team = "Middle East" }),
	} {
		t.Run(why, func(t *testing.T) {
			_, err := found.Check(story)

			require.ErrorIs(t, err, posts.ErrInvalidStory)
			assert.ErrorContains(t, err, why)
		})
	}
}

func with(story posts.Story, change func(*posts.Story)) posts.Story {
	change(&story)
	return story
}

func TestATourAPlaceAndASeasonalStoryAreTheirOwnAndKeyedAsTheGeneratorKeysThem(t *testing.T) {
	found := findings(t)

	tour, err := found.Check(posts.Story{Tour: 1})
	require.NoError(t, err)
	continent, err := found.Check(posts.Story{Place: 1})
	require.NoError(t, err)
	world, err := found.Check(posts.Story{Place: 2})
	require.NoError(t, err)
	seasonal, err := found.Check(posts.Story{Seasonal: 1})
	require.NoError(t, err)

	assert.Equal(t, posts.StoryKey{Flag: "fr", Continent: "the world"}, tour)
	assert.Equal(t, posts.StoryKey{Flag: "fr", Continent: "South Asia", Angle: posts.Holding}, continent)
	assert.Equal(t, posts.StoryKey{Flag: "pw", Continent: "the world", Angle: posts.Holding}, world)
	assert.Equal(t, day18, seasonal)
}

func TestAStoryNamesOneOfTheFindingsAndOneThatExists(t *testing.T) {
	found := findings(t)

	for why, story := range map[string]posts.Story{
		"names 0 of a front, a tour, a place and a seasonal story, not one": {},
		"names 2 of a front, a tour, a place and a seasonal story, not one": {Front: 1, Tour: 1},
		"no tour 3 of 1":           {Tour: 3},
		"no place 3 of 2":          {Place: 3},
		"no seasonal story 2 of 1": {Seasonal: 2},
		"no front 5 of 1":          {Front: 5},
	} {
		t.Run(why, func(t *testing.T) {
			_, err := found.Check(story)

			require.ErrorIs(t, err, posts.ErrInvalidStory)
			assert.ErrorContains(t, err, why)
		})
	}
}

func TestASeasonalStoryWithNoDayIsNoFindings(t *testing.T) {
	_, err := posts.FindingsOf(posts.Findings{Seasonal: []posts.Seasonal{{Title: "FRANCE WINS", Key: posts.StoryKey{Flag: "fr", Continent: "Europe"}}}})

	require.ErrorIs(t, err, posts.ErrInvalidFindings)
}

func TestWhatAStoryTellsOfIsSaidByItsNumber(t *testing.T) {
	assert.Equal(t, "front 3", posts.Story{Front: 3}.Names())
	assert.Equal(t, "tour 1", posts.Story{Tour: 1}.Names())
	assert.Equal(t, "place 2", posts.Story{Place: 2}.Names())
	assert.Equal(t, "seasonal 1", posts.Story{Seasonal: 1}.Names())
}
