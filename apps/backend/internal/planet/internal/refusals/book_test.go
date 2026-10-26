package refusals_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/refusals"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

const (
	scope   = "203.0.113.7"
	account = "01926c6e-7a4b-7c3d-8e9f-0a1b2c3d4e5f"
)

func TestAPayerIsRefusedUntilTheRefusalThatEndsLast(t *testing.T) {
	store := refusals.NewMemoryStore()
	book := refusals.NewBook(store, cptime.NewFixedClock(at))

	require.NoError(t, book.Decide(t.Context(), refusals.Caller{Scope: scope}, refusals.Refused(at.Add(time.Hour), at)))
	require.NoError(t, book.Decide(t.Context(), refusals.Caller{Account: account}, refusals.Refused(at.Add(24*time.Hour), at)))

	refusal, refused, err := book.Refusal(t.Context(), clicks.Payer{Scope: scope, Account: account})
	require.NoError(t, err)
	require.True(t, refused)
	assert.Equal(t, at.Add(24*time.Hour), refusal.Until())

	refusal, refused, err = book.Refusal(t.Context(), clicks.Payer{Scope: "198.51.100.4", Account: account})
	require.NoError(t, err)
	assert.True(t, refused, "an account is refused from any scope")
	assert.Equal(t, at.Add(24*time.Hour), refusal.Until())

	refusal, refused, err = book.Refusal(t.Context(), clicks.Payer{Scope: scope})
	require.NoError(t, err)
	assert.True(t, refused, "a scope is refused whatever account clicks from it")
	assert.Equal(t, at.Add(time.Hour), refusal.Until())
}

func TestARefusalThatEndedRefusesNothing(t *testing.T) {
	clock := cptime.NewFixedClock(at)
	book := refusals.NewBook(refusals.NewMemoryStore(), clock)
	require.NoError(t, book.Decide(t.Context(), refusals.Caller{Scope: scope}, refusals.Refused(at.Add(time.Hour), at)))

	clock.Advance(time.Hour)

	_, refused, err := book.Refusal(t.Context(), clicks.Payer{Scope: scope})
	require.NoError(t, err)
	assert.False(t, refused)
}

func TestAnAdmittedCallerIsRefusedNothing(t *testing.T) {
	clock := cptime.NewFixedClock(at)
	book := refusals.NewBook(refusals.NewMemoryStore(), clock)
	caller := refusals.Caller{Account: account}
	require.NoError(t, book.Decide(t.Context(), caller, refusals.Refused(at.Add(time.Hour), at)))

	clock.Advance(time.Minute)
	require.NoError(t, book.Decide(t.Context(), caller, refusals.Admitted(clock.Now())))
	require.NoError(t, book.Decide(t.Context(), caller, refusals.Refused(at.Add(time.Hour), at)))

	_, refused, err := book.Refusal(t.Context(), clicks.Payer{Scope: scope, Account: account})
	require.NoError(t, err)
	assert.False(t, refused, "the refusal sent again after the admission is older than it")
}

func TestARefusalThatCannotBeReadSaysSo(t *testing.T) {
	store := refusals.NewMemoryStore()
	store.FailWith(errors.New("postgres is down"))

	_, refused, err := refusals.NewBook(store, cptime.NewFixedClock(at)).Refusal(t.Context(), clicks.Payer{Scope: scope})

	require.Error(t, err)
	assert.False(t, refused)
}
