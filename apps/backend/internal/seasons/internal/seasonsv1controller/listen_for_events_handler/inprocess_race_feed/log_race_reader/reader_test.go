package log_race_reader_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	seasonsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/seasons/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/seasonsv1controller/listen_for_events_handler/inprocess_race_feed/log_race_reader"
)

type reader struct {
	err error
}

func (r reader) Race(context.Context) (*seasonsv1.Race, error) {
	return &seasonsv1.Race{}, r.err
}

func TestAFailedReadIsLogged(t *testing.T) {
	var out bytes.Buffer
	refused := errors.New("postgres is down")

	_, err := log_race_reader.New(reader{err: refused}, slog.New(slog.NewTextHandler(&out, nil))).Race(t.Context())

	require.ErrorIs(t, err, refused)
	assert.Contains(t, out.String(), `level=ERROR msg="failed to read the live race" error="postgres is down"`)
}

func TestAReadCutByTheShutdownLogsNothing(t *testing.T) {
	var out bytes.Buffer
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := log_race_reader.New(reader{err: context.Canceled}, slog.New(slog.NewTextHandler(&out, nil))).Race(ctx)

	require.ErrorIs(t, err, context.Canceled)
	assert.Empty(t, out.String())
}

func TestAReadLogsNothing(t *testing.T) {
	var out bytes.Buffer

	race, err := log_race_reader.New(reader{}, slog.New(slog.NewTextHandler(&out, nil))).Race(t.Context())

	require.NoError(t, err)
	assert.NotNil(t, race)
	assert.Empty(t, out.String())
}
