package acknowledge_notice_handler_test

import (
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	antibotv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/antibot/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/antibotv1controller/acknowledge_notice_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/notices"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

const (
	ada   = "0b7e5b6c-8f3a-4d2e-9c1a-2f6d8e4b7a10"
	grace = "0b7e5b6c-8f3a-4d2e-9c1a-2f6d8e4b7a11"
)

func TestTheCallersOwnNoticeIsAcknowledged(t *testing.T) {
	book := notices.NewBook(cptime.NewFixedClock(time.Date(2026, 11, 15, 20, 0, 0, 0, time.UTC)),
		notices.NewMemoryStore(), &notices.MemoryFeed{})
	_, _, err := book.Notify(t.Context(), ada, grace)
	require.NoError(t, err)

	_, err = acknowledge_notice_handler.New(book).AcknowledgeNotice(
		cpctx.AddAccountToContext(t.Context(), ada), connect.NewRequest(&antibotv1.AcknowledgeNoticeRequest{}))
	require.NoError(t, err)

	pending, err := book.Pending(t.Context(), ada)
	require.NoError(t, err)
	assert.False(t, pending)
}

func TestACallerWithNoAccountIsUnauthenticated(t *testing.T) {
	book := notices.NewBook(cptime.NewFixedClock(time.Now()), notices.NewMemoryStore(), &notices.MemoryFeed{})

	_, err := acknowledge_notice_handler.New(book).AcknowledgeNotice(
		t.Context(), connect.NewRequest(&antibotv1.AcknowledgeNoticeRequest{}))

	assert.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
}
