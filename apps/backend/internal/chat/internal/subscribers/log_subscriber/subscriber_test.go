package log_subscriber_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/subscribers/log_subscriber"
)

type handler struct {
	err error
}

func (h handler) Handle(context.Context, *authv1.AccountDeleted) error { return h.err }

func TestARefusedEventIsLoggedWithItsType(t *testing.T) {
	var out bytes.Buffer
	refused := errors.New("not an account id")

	err := log_subscriber.New(handler{err: refused}, slog.New(slog.NewTextHandler(&out, nil))).
		Handle(t.Context(), &authv1.AccountDeleted{})

	assert.ErrorIs(t, err, refused)
	assert.Contains(t, out.String(), "auth.v1.AccountDeleted")
}

func TestAHandledEventLogsNothing(t *testing.T) {
	var out bytes.Buffer

	err := log_subscriber.New(handler{}, slog.New(slog.NewTextHandler(&out, nil))).Handle(t.Context(), &authv1.AccountDeleted{})

	require.NoError(t, err)
	assert.Empty(t, out.String())
}
