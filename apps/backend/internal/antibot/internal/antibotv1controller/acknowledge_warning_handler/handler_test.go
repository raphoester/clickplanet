package acknowledge_warning_handler_test

import (
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	antibotv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/antibot/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/antibotv1controller/acknowledge_warning_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/warnings"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

const ada = "0b7e5b6c-8f3a-4d2e-9c1a-2f6d8e4b7a10"

type enforcing struct{}

func (enforcing) Enforcing() bool { return true }

func TestTheCallersOwnWarningIsAcknowledged(t *testing.T) {
	book := warnings.NewBook(enforcing{}, cptime.NewFixedClock(time.Date(2026, 11, 10, 20, 0, 0, 0, time.UTC)),
		warnings.NewMemoryStore(), &warnings.MemoryFeed{})
	_, err := book.Warn(t.Context(), []string{ada})
	require.NoError(t, err)

	_, err = acknowledge_warning_handler.New(book).AcknowledgeWarning(
		cpctx.AddAccountToContext(t.Context(), ada), connect.NewRequest(&antibotv1.AcknowledgeWarningRequest{}))
	require.NoError(t, err)

	pending, err := book.Pending(t.Context(), ada)
	require.NoError(t, err)
	assert.False(t, pending)
}

func TestACallerWithNoAccountIsUnauthenticated(t *testing.T) {
	book := warnings.NewBook(enforcing{}, cptime.NewFixedClock(time.Now()), warnings.NewMemoryStore(), &warnings.MemoryFeed{})

	_, err := acknowledge_warning_handler.New(book).AcknowledgeWarning(
		t.Context(), connect.NewRequest(&antibotv1.AcknowledgeWarningRequest{}))

	assert.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
}
