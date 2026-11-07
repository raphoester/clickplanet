package log_subscriber_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/subscribers/log_subscriber"
)

type handler struct {
	err error
}

func (h handler) Handle(context.Context, *playerv1.PlayerForgotten) error { return h.err }

func TestARefusedEventIsLoggedWithItsType(t *testing.T) {
	var out bytes.Buffer
	refused := errors.New("not an account id")

	err := log_subscriber.New(handler{err: refused}, slog.New(slog.NewTextHandler(&out, nil))).
		Handle(t.Context(), &playerv1.PlayerForgotten{})

	assert.ErrorIs(t, err, refused)
	assert.Contains(t, out.String(), "player.v1.PlayerForgotten")
}

func TestAHandledEventLogsNothing(t *testing.T) {
	var out bytes.Buffer

	err := log_subscriber.New(handler{}, slog.New(slog.NewTextHandler(&out, nil))).Handle(t.Context(), &playerv1.PlayerForgotten{})

	require.NoError(t, err)
	assert.Empty(t, out.String())
}

type batchHandler struct {
	err error
}

func (h batchHandler) HandleBatch(context.Context, []*playerv1.PlayerForgotten) error { return h.err }

func TestARefusedBatchIsLoggedWithItsTypeAndSize(t *testing.T) {
	var out bytes.Buffer
	refused := errors.New("the database is down")
	err := log_subscriber.NewBatch(batchHandler{err: refused}, slog.New(slog.NewTextHandler(&out, nil))).
		HandleBatch(t.Context(), []*playerv1.PlayerForgotten{{}, {}})
	assert.ErrorIs(t, err, refused)
	assert.Contains(t, out.String(), "player.v1.PlayerForgotten")
	assert.Contains(t, out.String(), "events=2")
}
