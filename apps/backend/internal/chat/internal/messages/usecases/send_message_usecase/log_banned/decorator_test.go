package log_banned_test

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages/usecases/send_message_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages/usecases/send_message_usecase/log_banned"
)

var raider = messages.AccountID{15: 1}

type refusing struct{ err error }

func (r refusing) Execute(context.Context, send_message_usecase.In) (messages.Message, error) {
	return messages.Message{}, r.err
}

func decorated(err error) (*log_banned.Decorator, *bytes.Buffer) {
	var out bytes.Buffer
	return log_banned.New(refusing{err: err}, slog.New(slog.NewTextHandler(&out, nil))), &out
}

func TestABannedPhraseIsLoggedWithItsSenderAndPassedOnAsAnInvalidMessage(t *testing.T) {
	cause := fmt.Errorf("failed to draft: %w: %w: %q", messages.ErrInvalidMessage, messages.ErrBanned, "sieg heil")
	decorator, out := decorated(cause)

	_, err := decorator.Execute(t.Context(), send_message_usecase.In{Account: raider, Room: messages.World, Text: "S1EG HEIL"})

	require.ErrorIs(t, err, messages.ErrInvalidMessage)
	assert.Contains(t, out.String(), "level=WARN")
	assert.Contains(t, out.String(), raider.String())
	assert.Contains(t, out.String(), `sieg heil`)
	assert.NotContains(t, out.String(), "S1EG HEIL", "the phrase listed, never the text")
}

func TestAnyOtherRefusalIsNotLogged(t *testing.T) {
	decorator, out := decorated(fmt.Errorf("%w: empty", messages.ErrInvalidMessage))

	_, err := decorator.Execute(t.Context(), send_message_usecase.In{Account: raider, Room: messages.World})

	require.ErrorIs(t, err, messages.ErrInvalidMessage)
	assert.Empty(t, out.String())
}

func TestAPostThatGoesIsNotLogged(t *testing.T) {
	decorator, out := decorated(nil)

	_, err := decorator.Execute(t.Context(), send_message_usecase.In{Account: raider, Room: messages.World})

	require.NoError(t, err)
	assert.Empty(t, out.String())
}
