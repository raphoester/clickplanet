package titles_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/titles"
)

var (
	first   = titles.FakeTitle{Key: "first", Tiles: 1}
	third   = titles.FakeTitle{Key: "third", Tiles: 3}
	catalog = titles.CatalogOf([]titles.Title{first, third})

	badge  = titles.FakeTitle{Key: "badge", Tiles: 0}
	low    = titles.FakeTitle{Key: "low", Tiles: 2}
	mid    = titles.FakeTitle{Key: "mid", Tiles: 5}
	high   = titles.FakeTitle{Key: "high", Tiles: 9}
	other  = titles.FakeTitle{Key: "other", Tiles: 4}
	ladder = titles.FakeTrack{Key: "ladder", Steps: []titles.Rank{low, mid, high}}
	side   = titles.FakeTrack{Key: "side", Steps: []titles.Rank{other}}
	tracks = titles.CatalogOf([]titles.Title{badge}, ladder, side)
)

type standingView struct {
	ID        titles.ID
	Track     titles.TrackID
	TrackName string
	Number    int
	Count     int
}

func alone(title titles.FakeTitle) standingView {
	return standingView{ID: title.Key}
}

func ranked(title titles.FakeTitle, track titles.FakeTrack, number int) standingView {
	return standingView{ID: title.Key, Track: track.Key, TrackName: track.Name(), Number: number, Count: len(track.Steps)}
}

func viewOf(standing titles.Standing) standingView {
	place := standing.Place()
	view := standingView{Track: place.Track(), TrackName: place.TrackName(), Number: place.Number(), Count: place.Count()}
	if !standing.Empty() {
		view.ID = standing.Title().ID()
	}
	return view
}

func viewsOf(standings []titles.Standing) []standingView {
	views := make([]standingView, 0, len(standings))
	for _, standing := range standings {
		views = append(views, viewOf(standing))
	}
	return views
}

type stepView struct {
	Standing  standingView
	Threshold uint64
	Earned    bool
}

type trackView struct {
	ID       titles.TrackID
	Name     string
	Progress uint64
	Steps    []stepView
}

func tracksOf(progress []titles.TrackProgress) []trackView {
	views := make([]trackView, 0, len(progress))
	for _, track := range progress {
		steps := make([]stepView, 0, len(track.Steps()))
		for _, step := range track.Steps() {
			steps = append(steps, stepView{Standing: viewOf(step.Standing()), Threshold: step.Threshold(), Earned: step.Earned()})
		}
		views = append(views, trackView{ID: track.ID(), Name: track.Name(), Progress: track.Progress(), Steps: steps})
	}
	return views
}

func TestTheCatalogNamesTheTitlesStatsEarnInItsOrder(t *testing.T) {
	assert.Empty(t, catalog.EarnedBy(titles.Career{}))
	assert.Equal(t, titles.IDs{"first"}, catalog.EarnedBy(tiles(2)))
	assert.Equal(t, titles.IDs{"third", "first"}, titles.CatalogOf([]titles.Title{third, first}).EarnedBy(tiles(3)))
	assert.Equal(t, titles.IDs{"badge", "low", "mid", "other"}, tracks.EarnedBy(tiles(5)), "standalone first, then each track in order")
}

func TestAGuestEarnsNothing(t *testing.T) {
	guest := titles.CareerOf(players.StatsOf(players.AccountID{}, 1_000_000, players.StreakOf(0, players.Day{}), 1_000, 1_000_000), players.Account{})

	assert.Empty(t, catalog.EarnedBy(guest))
	assert.Empty(t, titles.NewCatalog().EarnedBy(guest))
}

func TestTheReconciliationGrantsWhatIsEarnedAndRevokesWhatIsNot(t *testing.T) {
	career := func(account byte, taken uint64, linked bool) titles.Career {
		return titles.CareerOf(players.StatsOf(players.AccountID{15: account}, taken, players.StreakOf(0, players.Day{}), 0, 0), players.AccountOf(linked, time.Time{}))
	}

	reconciliation := catalog.ReconciliationOf([]titles.Career{
		career(1, 5, true),
		career(2, 1, true),
		career(3, 5, false),
		career(4, 0, true),
	}, titles.Holdings{
		{15: 1}: {"first"},
		{15: 2}: {"first", "third", "retired"},
		{15: 3}: {"first"},
	})

	assert.Equal(t, titles.Holdings{{15: 1}: {"third"}}, reconciliation.Grants())
	assert.Equal(t, titles.Holdings{{15: 2}: {"third", "retired"}, {15: 3}: {"first"}}, reconciliation.Revocations())
	assert.Equal(t, 3, reconciliation.Revocations().Len())
}

func TestATitleStandsAloneOrAtItsPlaceInItsTrack(t *testing.T) {
	standing, ok := tracks.StandingOf("badge")
	assert.True(t, ok)
	assert.Equal(t, alone(badge), viewOf(standing))
	assert.False(t, standing.Place().Ranked())

	standing, ok = tracks.StandingOf("mid")
	assert.True(t, ok)
	assert.Equal(t, ranked(mid, ladder, 2), viewOf(standing))

	_, ok = tracks.StandingOf("retired")
	assert.False(t, ok)
}

func TestShownIsEveryStandaloneTitleHeldAndTheHighestRankOfEachTrack(t *testing.T) {
	shown := tracks.Shown(titles.IDs{"low", "mid", "badge", "other", "retired"})

	assert.Equal(t, []standingView{alone(badge), ranked(mid, ladder, 2), ranked(other, side, 1)}, viewsOf(shown))
	assert.Empty(t, tracks.Shown(nil))
}

func TestProgressListsEachTracksRanksWhatIsHeldAndHowFarTheCareerIs(t *testing.T) {
	progress := tracks.Progress(tiles(6), titles.IDs{"low", "mid"})

	assert.Equal(t, []trackView{
		{ID: "ladder", Name: "LADDER", Progress: 6, Steps: []stepView{
			{Standing: ranked(low, ladder, 1), Threshold: 2, Earned: true},
			{Standing: ranked(mid, ladder, 2), Threshold: 5, Earned: true},
			{Standing: ranked(high, ladder, 3), Threshold: 9, Earned: false},
		}},
		{ID: "side", Name: "SIDE", Progress: 6, Steps: []stepView{
			{Standing: ranked(other, side, 1), Threshold: 4, Earned: false},
		}},
	}, tracksOf(progress), "a rank is earned only once granted, not because the career has reached it")
}

func TestWithoutLeavesOutTheHeldIDsAndKeepsTheOrder(t *testing.T) {
	earned := titles.IDs{"a", "b", "c"}

	assert.Equal(t, titles.IDs{"a", "c"}, earned.Without(titles.IDs{"b"}))
	assert.Empty(t, earned.Without(earned))
	assert.Equal(t, titles.IDs{"a", "b", "c"}, earned, "the receiver is left as it was")
}
