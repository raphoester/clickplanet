package log_board_reader_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	seasonsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/seasons/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/seasonsv1controller/listen_for_events_handler/inprocess_board_feed/log_board_reader"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/standings"
)

type reader struct {
	err error
}

func (r reader) Board(context.Context, string) (*seasonsv1.Board, []standings.AccountID, error) {
	return &seasonsv1.Board{}, []standings.AccountID{}, r.err
}

func TestAFailedReadIsLoggedWithItsCountry(t *testing.T) {
	var out bytes.Buffer
	refused := errors.New("postgres is down")

	_, _, err := log_board_reader.New(reader{err: refused}, slog.New(slog.NewTextHandler(&out, nil))).Board(t.Context(), "fr")

	require.ErrorIs(t, err, refused)
	assert.Contains(t, out.String(), "country=fr")
	assert.Contains(t, out.String(), "postgres is down")
}

func TestAReadCutByTheShutdownLogsNothing(t *testing.T) {
	var out bytes.Buffer
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, _, err := log_board_reader.New(reader{err: context.Canceled}, slog.New(slog.NewTextHandler(&out, nil))).Board(ctx, "")

	require.ErrorIs(t, err, context.Canceled)
	assert.Empty(t, out.String())
}

func TestAReadLogsNothing(t *testing.T) {
	var out bytes.Buffer

	board, _, err := log_board_reader.New(reader{}, slog.New(slog.NewTextHandler(&out, nil))).Board(t.Context(), "")

	require.NoError(t, err)
	assert.NotNil(t, board)
	assert.Empty(t, out.String())
}
