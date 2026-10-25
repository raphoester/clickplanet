package log_run_fills_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/fills"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/fills/usecases/run_fills_usecase/log_run_fills"
)

type stub struct {
	finished []fills.Fill
	err      error
}

func (s stub) Execute(context.Context) ([]fills.Fill, error) {
	return s.finished, s.err
}

var at = time.Date(2026, 10, 31, 21, 0, 0, 0, time.UTC)

func TestEachFillFinishedIsLoggedAndAFailureWithWhy(t *testing.T) {
	var out bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&out, nil))
	done, err := fills.Snapshot{
		ID: "season-0:finale:fill", Source: fills.SourceInternal, StartedAt: at, Filled: 1200, FinishedAt: at.Add(4 * time.Second),
	}.Restore()
	require.NoError(t, err)
	down := errors.New("postgres is down")

	_, err = log_run_fills.New(stub{finished: []fills.Fill{done}, err: down}, logger).Execute(t.Context())

	require.ErrorIs(t, err, down)
	assert.Contains(t, out.String(), `level=WARN msg="charges filled" request=season-0:finale:fill accounts=1200 took=4s`)
	assert.Contains(t, out.String(), `level=ERROR msg="failed to fill the charges" error="postgres is down"`)
}

func TestAFailureAtShutdownIsNotOne(t *testing.T) {
	var out bytes.Buffer
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := log_run_fills.New(stub{err: context.Canceled}, slog.New(slog.NewTextHandler(&out, nil))).Execute(ctx)

	require.ErrorIs(t, err, context.Canceled)
	assert.Empty(t, out.String())
}
