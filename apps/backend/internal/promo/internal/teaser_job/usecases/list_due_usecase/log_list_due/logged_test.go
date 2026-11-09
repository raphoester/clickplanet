package log_list_due_test

import (
	"bytes"
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/teaser_job"
	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/teaser_job/usecases/list_due_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/teaser_job/usecases/list_due_usecase/log_list_due"
)

type stub []list_due_usecase.Kept

func (s stub) Execute(context.Context) ([]list_due_usecase.Kept, error) {
	return s, nil
}

func run(t *testing.T, kept stub) string {
	t.Helper()
	var logs bytes.Buffer
	_, err := log_list_due.New(kept, slog.New(slog.NewTextHandler(&logs, nil))).Execute(t.Context())
	require.NoError(t, err)
	return logs.String()
}

func TestEachBattleDueIsLoggedAndNoneSaysWhy(t *testing.T) {
	start := time.Date(2026, 11, 14, 20, 0, 0, 0, time.UTC)

	assert.Contains(t, run(t, stub{{Folder: "season-1_finale", Due: teaser_job.Due{Battle: "season-1/finale", StartsAt: start, Cards: 2}}}),
		`level=INFO msg="a battle is due its teaser" battle=season-1/finale starts=2026-11-14T20:00:00.000Z cards=2 folder=season-1_finale`)
	assert.Contains(t, run(t, nil), `msg="no battle is due its teaser: none starts within 36 hours, or each is teased already"`)
}
