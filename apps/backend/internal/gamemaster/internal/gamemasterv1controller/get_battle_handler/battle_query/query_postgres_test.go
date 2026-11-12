package battle_query_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/gamemasterv1controller/get_battle_handler/battle_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/schedule"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/schedule/postgres_entry_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg/cppgtest"
)

var server cppgtest.Server

func TestMain(m *testing.M) { server.Run(m) }

var (
	starts = time.Date(2026, 11, 5, 20, 0, 0, 0, time.UTC)
	urf    = schedule.EntryID(uuid.UUID{15: 1})
	calm   = schedule.EntryID(uuid.UUID{15: 2})
)

func seeded(t *testing.T) *battle_query.PostgresQuery {
	t.Helper()
	db := server.CreateDatabase(t).OpenSchema(t, "gamemaster", migrations.FS)
	store := postgres_entry_store.New(db)
	for _, entry := range []schedule.EntrySnapshot{
		{ID: urf, Set: "urf", Kind: schedule.Battle, StartsAt: starts, EndsAt: starts.Add(time.Hour), Priority: 1, Times: 2, MadeAt: starts},
		{ID: calm, Set: "calm", Kind: schedule.Plain, StartsAt: starts, EndsAt: starts.Add(time.Hour), Priority: 1, MadeAt: starts},
	} {
		require.NoError(t, store.Save(t.Context(), entry.Restore()))
	}
	return battle_query.NewPostgresQuery(db)
}

func TestABattleEntryPlaysItsSetAndItsTimes(t *testing.T) {
	battle, err := seeded(t).Battle(t.Context(), urf.String())

	require.NoError(t, err)
	assert.Equal(t, "urf", battle.GetRuleSet())
	assert.Equal(t, uint32(2), battle.GetTimes())
}

func TestASeasonsFinalePlaysItsSetAndItsTimes(t *testing.T) {
	// Season 0's finale is seeded with rush, triple points.
	battle, err := seeded(t).Battle(t.Context(), "season-0/finale")

	require.NoError(t, err)
	assert.Equal(t, "rush", battle.GetRuleSet())
	assert.Equal(t, uint32(3), battle.GetTimes())
}

func TestWhatIsNoBattleIsNone(t *testing.T) {
	query := seeded(t)
	for _, id := range []string{
		calm.String(), // a plain entry
		schedule.EntryID(uuid.UUID{15: 9}).String(), // an entry not kept
		"season-1/finale", // a season not kept
		"season-0/day-3",  // a day
		"rush",            // nothing the timeline names
		"",
	} {
		_, err := query.Battle(t.Context(), id)
		require.ErrorIs(t, err, battle_query.ErrNoBattle, id)
	}
}
