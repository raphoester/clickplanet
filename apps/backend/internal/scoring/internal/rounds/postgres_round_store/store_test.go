package postgres_round_store_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/rounds"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/rounds/postgres_round_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(testSuite))
}

type testSuite struct {
	rounds.StoreContractSuite

	db *cppg.Postgres
}

func (s *testSuite) SetupSuite() {
	s.db = cppg.StartTestServer(s.T()).OpenSchema(s.T(), "scoring", migrations.FS)
	store := postgres_round_store.New(s.db)
	s.NewStore = func() rounds.Store { return store }
}

func (s *testSuite) SetupTest() {
	s.Require().NoError(s.db.Purge(s.T().Context()))
	s.StoreContractSuite.SetupTest()
}

func TestThePostgresStoreCountsTheTakesAsTheContractSays(t *testing.T) {
	db := cppg.StartTestServer(t).OpenSchema(t, "scoring", migrations.FS)
	store := postgres_round_store.New(db)

	suite.Run(t, &rounds.TakesContractSuite{
		NewStores: func() (rounds.Store, rounds.Takes) {
			require.NoError(t, db.Purge(t.Context()))
			return store, store
		},
		CountedIn: func(round rounds.Round) rounds.Counted { return countedIn(t, db, round) },
	})
}

func countedIn(t *testing.T, db *cppg.Postgres, round rounds.Round) rounds.Counted {
	t.Helper()

	counted := rounds.Counted{Countries: map[rounds.Country]uint64{}, Players: map[rounds.AccountID]map[rounds.Country]uint64{}}
	countries, err := db.On(t.Context()).QueryContext(t.Context(), `
		SELECT country, tiles FROM round_takes WHERE season = $1 AND ends_at = $2 AND own = $3
	`, int64(round.Season), round.EndsAt, round.Own)
	require.NoError(t, err)
	defer func() { _ = countries.Close() }()
	for countries.Next() {
		var (
			country string
			tiles   int64
		)
		require.NoError(t, countries.Scan(&country, &tiles))
		counted.Countries[rounds.Country(country)] = uint64(tiles) //nolint:gosec // CHECK (tiles > 0).
	}
	require.NoError(t, countries.Err())

	players, err := db.On(t.Context()).QueryContext(t.Context(), `
		SELECT account_id, country, tiles FROM round_players WHERE season = $1 AND ends_at = $2 AND own = $3
	`, int64(round.Season), round.EndsAt, round.Own)
	require.NoError(t, err)
	defer func() { _ = players.Close() }()
	for players.Next() {
		var (
			account uuid.UUID
			country string
			tiles   int64
		)
		require.NoError(t, players.Scan(&account, &country, &tiles))
		if counted.Players[rounds.AccountID(account)] == nil {
			counted.Players[rounds.AccountID(account)] = map[rounds.Country]uint64{}
		}
		counted.Players[rounds.AccountID(account)][rounds.Country(country)] = uint64(tiles) //nolint:gosec // CHECK (tiles > 0).
	}
	require.NoError(t, players.Err())
	return counted
}
