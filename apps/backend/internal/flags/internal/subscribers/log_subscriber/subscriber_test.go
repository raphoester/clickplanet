package log_subscriber_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	flagsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/flags/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/subscribers/log_subscriber"
)

type handler struct {
	err error
}

func (h handler) Handle(context.Context, *flagsv1.FlagOpened) error { return h.err }

func TestARefusedEventIsLoggedWithItsType(t *testing.T) {
	var out bytes.Buffer
	refused := errors.New("not an account id")

	err := log_subscriber.New(handler{err: refused}, slog.New(slog.NewTextHandler(&out, nil))).
		Handle(t.Context(), &flagsv1.FlagOpened{Flag: "r-0000000001"})

	assert.ErrorIs(t, err, refused)
	assert.Contains(t, out.String(), "flags.v1.FlagOpened")
}

func TestAHandledEventLogsNothing(t *testing.T) {
	var out bytes.Buffer

	err := log_subscriber.New(handler{}, slog.New(slog.NewTextHandler(&out, nil))).Handle(t.Context(), &flagsv1.FlagOpened{Flag: "r-0000000001"})

	require.NoError(t, err)
	assert.Empty(t, out.String())
}
