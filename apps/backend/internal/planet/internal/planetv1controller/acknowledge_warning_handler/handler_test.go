package acknowledge_warning_handler_test

import (
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/acknowledge_warning_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/refusals"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

const ada = "01926c6e-7a4b-7c3d-8e9f-0a1b2c3d4e5f"

var at = time.Date(2026, 11, 10, 20, 0, 0, 0, time.UTC)

func TestTheCallersOwnWarningIsAcknowledged(t *testing.T) {
	book := refusals.NewBook(refusals.NewMemoryStore(), cptime.NewFixedClock(at.Add(time.Minute)))
	require.NoError(t, book.Decide(t.Context(), refusals.Caller{Account: ada}, refusals.Warned(at)))

	_, err := acknowledge_warning_handler.New(book).AcknowledgeWarning(
		cpctx.AddAccountToContext(t.Context(), ada), connect.NewRequest(&planetv1.AcknowledgeWarningRequest{}))
	require.NoError(t, err)

	_, refused, err := book.Refusal(t.Context(), clicks.Payer{Account: ada})
	require.NoError(t, err)
	assert.False(t, refused)
}
