package warn_caller_handler_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/commands/warn_caller_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/refusals"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var at = time.Date(2026, 11, 10, 20, 0, 0, 0, time.UTC)

const account = "01926c6e-7a4b-7c3d-8e9f-0a1b2c3d4e5f"

func warn(decided time.Time) *planetv1.WarnCaller {
	return &planetv1.WarnCaller{AccountId: account, DecidedAt: timestamppb.New(decided)}
}

func TestAWarnedAccountIsRefusedUntilItAcknowledges(t *testing.T) {
	book := refusals.NewBook(refusals.NewMemoryStore(), cptime.NewFixedClock(at.Add(time.Hour)))

	require.NoError(t, warn_caller_handler.New(book).Handle(t.Context(), warn(at)))

	refusal, refused, err := book.Refusal(t.Context(), clicks.Payer{Scope: "203.0.113.7", Account: account})
	require.NoError(t, err)
	require.True(t, refused)
	assert.True(t, refusal.Warning())
}

func TestAWarningSentAgainAfterItsAcknowledgementChangesNothing(t *testing.T) {
	store := refusals.NewMemoryStore()
	book := refusals.NewBook(store, cptime.NewFixedClock(at.Add(time.Minute)))
	handler := warn_caller_handler.New(book)
	payer := clicks.Payer{Account: account}

	require.NoError(t, handler.Handle(t.Context(), warn(at)))
	require.NoError(t, book.Acknowledge(t.Context(), payer))
	require.NoError(t, handler.Handle(t.Context(), warn(at)))

	_, refused, err := book.Refusal(t.Context(), payer)
	require.NoError(t, err)
	assert.False(t, refused)
}

func TestACommandThatNamesNoAccountIsRefused(t *testing.T) {
	handler := warn_caller_handler.New(refusals.NewBook(refusals.NewMemoryStore(), cptime.NewFixedClock(at)))

	err := handler.Handle(t.Context(), &planetv1.WarnCaller{DecidedAt: timestamppb.New(at)})

	assert.ErrorIs(t, err, refusals.ErrNoCaller)
}
