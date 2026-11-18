package audit_forgive_missed_day_test

import (
	"bytes"
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/stats"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/stats/usecases/forgive_missed_day_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/stats/usecases/forgive_missed_day_usecase/audit_forgive_missed_day"
)

var (
	ada  = accounts.AccountID{15: 1}
	noon = time.Date(2026, 11, 17, 12, 0, 0, 0, time.UTC)
	in   = forgive_missed_day_usecase.In{Account: ada, Day: stats.DayOf(noon.AddDate(0, 0, -2))}
)

type stubExecutor struct {
	forgiven forgive_missed_day_usecase.Forgiven
	err      error
}

func (s stubExecutor) Execute(context.Context, forgive_missed_day_usecase.In) (forgive_missed_day_usecase.Forgiven, error) {
	return s.forgiven, s.err
}

func run(t *testing.T, inner stubExecutor, in forgive_missed_day_usecase.In) (string, error) {
	t.Helper()

	var logs bytes.Buffer
	_, err := audit_forgive_missed_day.New(inner, slog.New(slog.NewTextHandler(&logs, nil))).Execute(t.Context(), in)
	return logs.String(), err
}

func streak(days uint32, best uint32) stats.Stats {
	return stats.StatsSnapshot{Streak: stats.StreakOf(days, stats.DayOf(noon)), StreakBest: best}.Restore()
}

func TestEveryForgivenDayIsLoggedWithTheStreakBeforeAndAfter(t *testing.T) {
	logs, err := run(t, stubExecutor{forgiven: forgive_missed_day_usecase.Forgiven{Before: streak(2, 37), After: streak(40, 40)}}, in)

	require.NoError(t, err)
	assert.Contains(t, logs, `level=WARN msg="admin missed day forgiven" account_id=`+ada.String()+
		` day=2026-11-15 dry_run=false before.streak=2 before.last_day=2026-11-17 before.best=37`+
		` after.streak=40 after.last_day=2026-11-17 after.best=40`)
}

func TestARefusalIsLoggedWithTheStreakBefore(t *testing.T) {
	dryRun := in
	dryRun.DryRun = true

	logs, err := run(t, stubExecutor{forgiven: forgive_missed_day_usecase.Forgiven{Before: streak(2, 37)}, err: stats.ErrDayPlayed}, dryRun)

	require.ErrorIs(t, err, stats.ErrDayPlayed)
	assert.Contains(t, logs, `level=WARN msg="admin missed day forgiveness failed" account_id=`+ada.String()+
		` day=2026-11-15 dry_run=true before.streak=2 before.last_day=2026-11-17 before.best=37 error="the account played that day"`)
}
