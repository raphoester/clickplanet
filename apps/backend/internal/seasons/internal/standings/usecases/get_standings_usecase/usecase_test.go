package get_standings_usecase_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/calendar"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/standings"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/standings/inmemory_contribution_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/standings/usecases/get_standings_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcountries"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var (
	seasonZeroEnds = time.Date(2026, 10, 31, 23, 0, 0, 0, time.UTC)
	ada            = standings.AccountID{0: 1, 15: 1}
	bob            = standings.AccountID{0: 1, 15: 2}
)

func twoSeasons() calendar.Calendar {
	return calendar.New(calendar.Config{List: []calendar.Entry{
		{Number: 0, EndsAt: seasonZeroEnds, Finale: 2 * time.Hour},
		{Number: 1, EndsAt: seasonZeroEnds.AddDate(0, 2, 0), Finale: 2 * time.Hour},
	}})
}

type setup struct {
	store   *inmemory_contribution_store.Store
	players *standings.FakePlayers
	clock   *cptime.FixedClock
	useCase *get_standings_usecase.UseCase
}

func newSetup(t *testing.T) setup {
	t.Helper()

	store := inmemory_contribution_store.New()
	players := standings.NewFakePlayers()
	players.Add(ada, standings.Player{Name: "Ada"})
	players.Add(bob, standings.Player{Name: "Bob"})
	require.NoError(t, store.RecordTake(t.Context(), 0, standings.Take{Account: ada, Country: "fr"}))
	require.NoError(t, store.RecordTake(t.Context(), 1, standings.Take{Account: bob, Country: "de"}))
	clock := cptime.NewFixedClock(seasonZeroEnds.Add(-time.Hour))

	return setup{
		store: store, players: players, clock: clock,
		useCase: get_standings_usecase.New(twoSeasons(), clock, cpcountries.New(), standings.NewBoard(store, players)),
	}
}

func names(top []standings.Standing) []string {
	found := make([]string, len(top))
	for i, standing := range top {
		found[i] = standing.Player.Name
	}
	return found
}

func TestTheTopIsTheCurrentSeasons(t *testing.T) {
	s := newSetup(t)

	top, err := s.useCase.Execute(t.Context(), "")
	require.NoError(t, err)
	assert.Equal(t, []string{"Ada"}, names(top))

	s.clock.Advance(2 * time.Hour)
	top, err = s.useCase.Execute(t.Context(), "")
	require.NoError(t, err)
	assert.Equal(t, []string{"Bob"}, names(top))
}

func TestTheTopOfACountry(t *testing.T) {
	s := newSetup(t)

	top, err := s.useCase.Execute(t.Context(), "fr")
	require.NoError(t, err)
	assert.Equal(t, []string{"Ada"}, names(top))

	top, err = s.useCase.Execute(t.Context(), "de")
	require.NoError(t, err)
	assert.Empty(t, top)
}

func TestNoSeasonIsAnEmptyTop(t *testing.T) {
	s := newSetup(t)
	s.clock.Advance(365 * 24 * time.Hour)

	top, err := s.useCase.Execute(t.Context(), "")

	require.NoError(t, err)
	assert.Empty(t, top)
}

func TestACountryThatIsNotOneIsRefusedAndReadsNothing(t *testing.T) {
	s := newSetup(t)

	_, err := s.useCase.Execute(t.Context(), "zz")

	require.ErrorIs(t, err, standings.ErrUnknownCountry)
	assert.Zero(t, s.players.Asked())
}

func TestAFailureToReadTheTopIsAnError(t *testing.T) {
	s := newSetup(t)
	refused := errors.New("the database is down")
	s.store.FailWith(refused)

	_, err := s.useCase.Execute(t.Context(), "")

	assert.ErrorIs(t, err, refused)
}
