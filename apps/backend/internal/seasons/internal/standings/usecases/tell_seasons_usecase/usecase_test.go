package tell_seasons_usecase_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	seasonsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/seasons/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/calendar"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/rounds"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/rounds/inmemory_round_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/standings"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/standings/inmemory_contribution_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/standings/usecases/tell_seasons_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpbootstrap"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var (
	end    = time.Date(2026, 10, 31, 23, 0, 0, 0, time.UTC)
	day    = rounds.Round{EndsAt: time.Date(2026, 10, 30, 21, 0, 0, 0, time.UTC)}
	finale = rounds.Round{EndsAt: end, Finale: true}

	ada = standings.AccountID{15: 1}
	bob = standings.AccountID{15: 2}
	cy  = standings.AccountID{15: 3}
)

type fixture struct {
	rounds  *inmemory_round_store.Store
	scores  *inmemory_contribution_store.Store
	events  *cpbootstrap.RecordedEvents
	clock   *cptime.FixedClock
	useCase *tell_seasons_usecase.UseCase
}

func newFixture(t *testing.T) fixture {
	t.Helper()

	f := fixture{
		rounds: inmemory_round_store.New(),
		scores: inmemory_contribution_store.New(),
		events: cpbootstrap.NewRecordedEvents(),
		clock:  cptime.NewFixedClock(end.Add(-time.Minute)),
	}
	f.useCase = tell_seasons_usecase.New(f.rounds, f.scores, f.events, calendar.New(calendar.Config{List: []calendar.Entry{
		{Number: 0, EndsAt: end, Finale: 2 * time.Hour},
		{Number: 1, EndsAt: end.Add(30 * 24 * time.Hour), Finale: 2 * time.Hour},
	}}), f.clock)

	f.take(t, 0, ada, "fr", 3)
	f.take(t, 0, bob, "de", 2)
	f.take(t, 0, cy, "fr", 1)
	f.take(t, 1, ada, "it", 1)
	f.close(t, day, rounds.Result{Country: "de", Rank: 1, Points: 25}, rounds.Result{Country: "fr", Rank: 2, Points: 18})
	return f
}

func (f fixture) take(t *testing.T, season calendar.Number, account standings.AccountID, country standings.Country, tiles int) {
	t.Helper()
	for range tiles {
		require.NoError(t, f.scores.RecordTake(t.Context(), season, standings.Take{Account: account, Country: country, At: end.Add(-time.Hour)}))
	}
}

func (f fixture) close(t *testing.T, round rounds.Round, results ...rounds.Result) {
	t.Helper()
	require.NoError(t, f.rounds.RecordSnapshot(t.Context(), round, rounds.Snapshot{Tiles: 10}))
	require.NoError(t, f.rounds.Close(t.Context(), round, results))
}

func (f fixture) closeTheFinale(t *testing.T) {
	t.Helper()
	f.close(t, finale, rounds.Result{Country: "fr", Rank: 1, Points: 75}, rounds.Result{Country: "de", Rank: 2, Points: 54})
}

func TestNothingIsToldWhileTheSeasonRuns(t *testing.T) {
	f := newFixture(t)

	told, err := f.useCase.Execute(t.Context(), nil)

	require.NoError(t, err)
	assert.Empty(t, told)
	assert.Empty(t, f.events.Published())
}

func TestASeasonOverWhoseFinaleIsStillOpenWaits(t *testing.T) {
	f := newFixture(t)
	f.clock.Advance(time.Minute + time.Second)

	told, err := f.useCase.Execute(t.Context(), nil)

	require.NoError(t, err)
	assert.Empty(t, told, "the finale closes on the first snapshot after the end")
	assert.Empty(t, f.events.Published())
}

func TestOnceTheFinaleClosedEachPlayerIsToldItsSeason(t *testing.T) {
	f := newFixture(t)
	f.clock.Advance(2 * time.Minute)
	f.closeTheFinale(t)

	told, err := f.useCase.Execute(t.Context(), nil)

	require.NoError(t, err)
	assert.Equal(t, []tell_seasons_usecase.Told{{Season: 0, Players: 3}}, told)
	assert.Equal(t, []proto.Message{
		&seasonsv1.SeasonPlayed{Season: 0, AccountId: ada.String(), MainFlag: "fr", Tiles: 3, Won: true},
		&seasonsv1.SeasonPlayed{Season: 0, AccountId: bob.String(), MainFlag: "de", Tiles: 2, Won: false},
		&seasonsv1.SeasonPlayed{Season: 0, AccountId: cy.String(), MainFlag: "fr", Tiles: 1, Won: true},
	}, f.events.Published(), "93 points to 79: France is the champion")
}

func TestASeasonToldAlreadyIsNotToldAgain(t *testing.T) {
	f := newFixture(t)
	f.clock.Advance(2 * time.Minute)
	f.closeTheFinale(t)

	told, err := f.useCase.Execute(t.Context(), []calendar.Number{0})

	require.NoError(t, err)
	assert.Empty(t, told)
	assert.Empty(t, f.events.Published())
}

func TestAFailedReadTellsNothingAndTheNextOneTellsTheSeason(t *testing.T) {
	f := newFixture(t)
	f.clock.Advance(2 * time.Minute)
	f.closeTheFinale(t)
	f.scores.FailWith(errors.New("postgres is down"))

	told, err := f.useCase.Execute(t.Context(), nil)

	require.Error(t, err)
	assert.Empty(t, told)
	assert.Empty(t, f.events.Published())

	f.scores.FailWith(nil)
	told, err = f.useCase.Execute(t.Context(), nil)

	require.NoError(t, err)
	assert.Equal(t, []tell_seasons_usecase.Told{{Season: 0, Players: 3}}, told)
	assert.Len(t, f.events.Published(), 3)
}
