package log_turn_reader_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	metagamev1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/metagame/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/metagame/internal/metagamev1controller/listen_for_events_handler/inprocess_turn_feed/log_turn_reader"
)

type reader struct {
	err error
}

func (r reader) Turn(context.Context) (*metagamev1.Turn, error) {
	return &metagamev1.Turn{}, r.err
}

func TestAFailedReadIsLogged(t *testing.T) {
	var out bytes.Buffer
	down := errors.New("postgres is down")

	_, err := log_turn_reader.New(reader{err: down}, slog.New(slog.NewTextHandler(&out, nil))).Turn(t.Context())

	require.ErrorIs(t, err, down)
	assert.Contains(t, out.String(), `level=ERROR msg="failed to read the live turn" error="postgres is down"`)
}

func TestAReadCutByTheShutdownLogsNothing(t *testing.T) {
	var out bytes.Buffer
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := log_turn_reader.New(reader{err: context.Canceled}, slog.New(slog.NewTextHandler(&out, nil))).Turn(ctx)

	require.ErrorIs(t, err, context.Canceled)
	assert.Empty(t, out.String())
}

func TestAReadLogsNothing(t *testing.T) {
	var out bytes.Buffer

	turn, err := log_turn_reader.New(reader{}, slog.New(slog.NewTextHandler(&out, nil))).Turn(t.Context())

	require.NoError(t, err)
	assert.NotNil(t, turn)
	assert.Empty(t, out.String())
}
