package migrations_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	missionsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/missions/v1"
	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	scoringv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/scoring/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpoutbox"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

const (
	briefsPerAccount = "20261028120000"
	countedFacts     = "20261028200000"
	gamemasterDays   = "20261029110000"
	redrawConquests  = "20261102070000"
)

func TestEachRowKeptBeforeGamemasterDaysNamesItsDayByItsRound(t *testing.T) {
	db := cppg.StartTestServer(t).OpenSchema(t, "missions", cppg.MigrationsBefore(t, migrations.FS, gamemasterDays))
	ada, mission := uuid.UUID{15: 1}, uuid.UUID{15: 9}
	seed := func(query string, args ...any) {
		_, err := db.On(t.Context()).ExecContext(t.Context(), query, args...)
		require.NoError(t, err)
	}
	seed(`INSERT INTO mission_briefs (day, account_id, missions) VALUES ('2026-10-29', $1, '[]')`, ada)
	seed(`INSERT INTO mission_sheets (day, account_id, counted_takes_below) VALUES ('2026-10-29', $1, 7)`, ada)
	seed(`INSERT INTO mission_progress (day, account_id, mission_id, progress) VALUES ('2026-10-29', $1, $2, 3)`, ada, mission)
	seed(`INSERT INTO mission_payments (day, account_id, mission_id) VALUES ('2026-10-29', $1, $2)`, ada, mission)

	require.NoError(t, db.Migrate(t.Context(), migrations.FS))

	var briefs, sheets, progress, payments string
	require.NoError(t, db.On(t.Context()).QueryRowContext(t.Context(), `
		SELECT (SELECT day FROM mission_briefs), (SELECT day FROM mission_sheets), (SELECT day FROM mission_progress),
		       (SELECT day FROM mission_payments)`).Scan(&briefs, &sheets, &progress, &payments))
	assert.Equal(t, []string{"season-0/day-22", "season-0/day-22", "season-0/day-22", "season-0/day-22"},
		[]string{briefs, sheets, progress, payments}, "day 22 of season 0 ended on 29 Oct")

	seed(`DELETE FROM mission_sheets`)
	var left int
	require.NoError(t, db.On(t.Context()).QueryRowContext(t.Context(),
		`SELECT (SELECT count(*) FROM mission_progress) + (SELECT count(*) FROM mission_payments)`).Scan(&left))
	assert.Zero(t, left, "a sheet's progress and payments still go with it")
}

func TestASheetKeptBeforeCountedFactsKeepsItsTakesAndCountedNothingElse(t *testing.T) {
	db := cppg.StartTestServer(t).OpenSchema(t, "missions", cppg.MigrationsBefore(t, migrations.FS, countedFacts))
	ada := uuid.UUID{15: 1}
	_, err := db.On(t.Context()).ExecContext(t.Context(),
		`INSERT INTO mission_sheets (day, account_id, counted_takes_below) VALUES ('2026-10-28', $1, 7)`, ada)
	require.NoError(t, err)

	require.NoError(t, db.Migrate(t.Context(), migrations.FS))

	var takes, bombs, shields, fortifies int64
	var boxed, answered bool
	require.NoError(t, db.On(t.Context()).QueryRowContext(t.Context(), `
		SELECT counted_takes_below, counted_bombs_below, counted_shields_below, counted_fortifies_below,
		       counted_boxes_until IS NOT NULL, counted_answers_until IS NOT NULL
		FROM mission_sheets WHERE account_id = $1`, ada).Scan(&takes, &bombs, &shields, &fortifies, &boxed, &answered))
	assert.Equal(t, []int64{7, 0, 0, 0}, []int64{takes, bombs, shields, fortifies})
	assert.False(t, boxed || answered)
}

func TestEachSheetKeptBeforeBriefsPerAccountKeepsItsFlagsBrief(t *testing.T) {
	db := cppg.StartTestServer(t).OpenSchema(t, "missions", cppg.MigrationsBefore(t, migrations.FS, briefsPerAccount))
	ada, mission := uuid.UUID{15: 1}, uuid.UUID{15: 9}
	seed := func(query string, args ...any) {
		_, err := db.On(t.Context()).ExecContext(t.Context(), query, args...)
		require.NoError(t, err)
	}
	seed(`INSERT INTO mission_briefs (day, flag, missions) VALUES ('2026-10-28', 'fr', '[{"id": "fr"}]'), ('2026-10-28', 'de', '[]')`)
	seed(`INSERT INTO mission_sheets (day, account_id, flag, counted_takes_below) VALUES ('2026-10-28', $1, 'fr', 7)`, ada)
	seed(`INSERT INTO mission_progress (day, account_id, mission_id, progress) VALUES ('2026-10-28', $1, $2, 3)`, ada, mission)

	require.NoError(t, db.Migrate(t.Context(), migrations.FS))

	var kept string
	require.NoError(t, db.On(t.Context()).QueryRowContext(t.Context(),
		`SELECT missions::text FROM mission_briefs WHERE day = 'season-0/day-21' AND account_id = $1`, ada).Scan(&kept))
	assert.JSONEq(t, `[{"id": "fr"}]`, kept)
	var briefs, progress int
	require.NoError(t, db.On(t.Context()).QueryRowContext(t.Context(),
		`SELECT (SELECT count(*) FROM mission_briefs), (SELECT count(*) FROM mission_progress)`).Scan(&briefs, &progress))
	assert.Equal(t, 1, briefs, "Germany's brief has no player")
	assert.Equal(t, 1, progress, "the progress stays")
}

func TestTheConquestsNotDoneOfTheBrokenDayGoAndTheirDaysKeepTheirCounts(t *testing.T) {
	db := cppg.StartTestServer(t).OpenSchema(t, "missions", cppg.MigrationsBefore(t, migrations.FS, redrawConquests))
	seed := func(query string, args ...any) {
		_, err := db.On(t.Context()).ExecContext(t.Context(), query, args...)
		require.NoError(t, err)
	}
	seed(`INSERT INTO mission_days (day_id, starts_at, ends_at) VALUES
		('season-1/day-2', '2026-11-01 21:00:00+00', '2026-11-02 21:00:00+00'),
		('season-1/day-1', '2026-10-31 21:00:00+00', '2026-11-01 21:00:00+00')`)
	for n, row := range []struct {
		day, flag, kind string
		done            bool
	}{
		{"season-1/day-2", "fr", "take_in", false},
		{"season-1/day-2", "de", "take_in", true},
		{"season-1/day-2", "dz", "break_shields", false},
		{"season-1/day-1", "fr", "take_in", false},
	} {
		seed(`INSERT INTO flag_days (day, flag, takes, counted_takes_below, counted_takes_below_index, strikes,
				counted_strikes_below, counted_strikes_below_index) VALUES ($1, $2, 40, 41, 1, 3, 4, 1)`, row.day, row.flag)
		seed(`INSERT INTO flag_missions (day, flag, mission_id, kind, payload, points, progress, done)
			VALUES ($1, $2, $3, $4::flag_mission_kind, '{}', 5, 12, $5)`, row.day, row.flag, uuid.UUID{15: byte(n + 1)}, row.kind, row.done)
	}

	require.NoError(t, db.Migrate(t.Context(), migrations.FS))

	rows, err := db.On(t.Context()).QueryContext(t.Context(), `SELECT day || '/' || flag FROM flag_missions ORDER BY day, flag`)
	require.NoError(t, err)
	defer func() { _ = rows.Close() }()
	var missions []string
	for rows.Next() {
		var mission string
		require.NoError(t, rows.Scan(&mission))
		missions = append(missions, mission)
	}
	require.NoError(t, rows.Err())
	assert.Equal(t, []string{"season-1/day-1/fr", "season-1/day-2/de", "season-1/day-2/dz"}, missions,
		"France's conquest of the day that ran on 2 Nov at 06:00 goes; one done, a shield mission and a day before stay")

	var days, takes int
	require.NoError(t, db.On(t.Context()).QueryRowContext(t.Context(),
		`SELECT count(*), sum(takes) FROM flag_days`).Scan(&days, &takes))
	assert.Equal(t, []int{4, 160}, []int{days, takes}, "every day keeps what it counted")
}

func TestTheOutboxKeepsEveryMessageMissionsSends(t *testing.T) {
	db := cppg.StartTestServer(t).OpenSchema(t, "missions", migrations.FS)

	require.NoError(t, db.InTx(t.Context(), func(ctx context.Context) error {
		return cpoutbox.New(db).Record(ctx, &planetv1.GrantCharges{}, &scoringv1.GrantPoints{}, &missionsv1.MissionsPaid{},
			&missionsv1.FlagMissionDone{})
	}))
}
