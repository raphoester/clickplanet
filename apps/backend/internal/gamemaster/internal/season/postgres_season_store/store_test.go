package postgres_season_store_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/season"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/season/postgres_season_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func TestThePostgresStoreKeepsTheContract(t *testing.T) {
	db := cppg.StartTestServer(t).OpenSchema(t, "gamemaster", migrations.FS)

	suite.Run(t, &season.StoreContractSuite{
		NewStore: func() season.Store {
			require.NoError(t, db.Purge(t.Context()))
			return postgres_season_store.New(db)
		},
	})
}

func TestSeasonZeroIsSeededWithItsFinalBattleOn31OctoberFrom21To23(t *testing.T) {
	db := cppg.StartTestServer(t).OpenSchema(t, "gamemaster", migrations.FS)

	calendar, err := postgres_season_store.New(db).Calendar(t.Context())

	require.NoError(t, err)
	windows := calendar.Windows()
	require.Len(t, windows, 1)
	assert.Equal(t, season.Finale{Length: 2 * time.Hour, Set: "rush", Description: "Triple points.", Times: 3}, windows[0].Finale)
	assert.Equal(t, time.Date(2026, 10, 31, 21, 0, 0, 0, time.UTC), windows[0].FinaleStartsAt())
	assert.Equal(t, time.Date(2026, 10, 31, 23, 0, 0, 0, time.UTC), windows[0].EndsAt)
}
