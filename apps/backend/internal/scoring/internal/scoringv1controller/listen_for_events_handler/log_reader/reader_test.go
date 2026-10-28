package log_reader_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	scoringv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/scoring/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/scoringv1controller/listen_for_events_handler/log_reader"
)

type reader struct {
	err error
}

func (r reader) Live(context.Context, string) (*scoringv1.Board, bool, error) {
	return &scoringv1.Board{}, r.err == nil, r.err
}

func TestAFailedReadIsLoggedWithItsKey(t *testing.T) {
	var out bytes.Buffer
	refused := errors.New("postgres is down")

	_, _, err := log_reader.New[string, *scoringv1.Board]("board", reader{err: refused}, slog.New(slog.NewTextHandler(&out, nil))).
		Live(t.Context(), "fr")

	require.ErrorIs(t, err, refused)
	assert.Contains(t, out.String(), "key=fr")
	assert.Contains(t, out.String(), "live board")
	assert.Contains(t, out.String(), "postgres is down")
}

func TestAReadCutByTheShutdownLogsNothing(t *testing.T) {
	var out bytes.Buffer
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, _, err := log_reader.New[string, *scoringv1.Board]("board", reader{err: context.Canceled}, slog.New(slog.NewTextHandler(&out, nil))).
		Live(ctx, "")

	require.ErrorIs(t, err, context.Canceled)
	assert.Empty(t, out.String())
}

func TestAReadLogsNothing(t *testing.T) {
	var out bytes.Buffer

	board, shows, err := log_reader.New[string, *scoringv1.Board]("board", reader{}, slog.New(slog.NewTextHandler(&out, nil))).
		Live(t.Context(), "")

	require.NoError(t, err)
	assert.True(t, shows)
	assert.NotNil(t, board)
	assert.Empty(t, out.String())
}
