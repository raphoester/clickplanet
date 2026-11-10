package take_snapshot_usecase_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/raphoester/clickplanet.lol-backend/internal/publishing/internal/stats"
	"github.com/raphoester/clickplanet.lol-backend/internal/publishing/internal/stats/usecases/take_snapshot_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type snapshots struct {
	taken  int
	failed bool
	latest time.Time
}

func (s *snapshots) Execute(context.Context) (stats.Report, error) {
	s.taken++
	if s.failed {
		return stats.Report{}, errors.New("zernio answered 500")
	}
	return stats.Report{}, nil
}

func (s *snapshots) Latest(context.Context) (time.Time, error) {
	return s.latest, nil
}

var due = time.Date(2026, 11, 10, 8, 17, 0, 0, time.UTC)

func runner(s *snapshots, clock *cptime.FixedClock) *take_snapshot_usecase.Runner {
	return take_snapshot_usecase.NewRunner(s, s, clock, slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)))
}

func TestTheDaysSnapshotIsTakenOnceItIsDueAndNoneWasTakenSince(t *testing.T) {
	clock := cptime.NewFixedClock(due.Add(-time.Minute))
	taken := &snapshots{latest: due.Add(-24 * time.Hour)}
	run := runner(taken, clock)

	run.Once(t.Context())
	assert.Zero(t, taken.taken, "yesterday's is the last one due")

	clock.Advance(2 * time.Hour)
	run.Once(t.Context())
	assert.Equal(t, 1, taken.taken, "a restart catches up the same day")

	taken.latest = clock.Now()
	run.Once(t.Context())
	assert.Equal(t, 1, taken.taken, "one a day")
}

func TestAFailedSnapshotIsTriedAgainAnHourLater(t *testing.T) {
	clock := cptime.NewFixedClock(due)
	failing := &snapshots{failed: true}
	run := runner(failing, clock)

	run.Once(t.Context())
	clock.Advance(50 * time.Minute)
	run.Once(t.Context())
	assert.Equal(t, 1, failing.taken)

	clock.Advance(10 * time.Minute)
	run.Once(t.Context())
	assert.Equal(t, 2, failing.taken)
}
