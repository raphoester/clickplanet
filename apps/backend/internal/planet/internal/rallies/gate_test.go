package rallies_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/allegiances"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rallies"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpflags"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type flags struct{ err error }

func (f flags) PlayError(context.Context, clicks.Payer, string) error { return f.err }

type refusals struct {
	refusal clicks.Refusal
	refused bool
	err     error
}

func (r refusals) Refusal(context.Context, clicks.Payer) (clicks.Refusal, bool, error) {
	return r.refusal, r.refused, r.err
}

type gateParts struct {
	flags    flags
	refusals refusals
	store    *rallies.MemoryStore
	clock    *cptime.FixedClock
}

func newGateParts() gateParts {
	return gateParts{store: rallies.NewMemoryStore(), clock: cptime.NewFixedClock(calledAt.Add(time.Minute))}
}

func (p gateParts) call(t *testing.T, country string) error {
	t.Helper()
	gate := rallies.NewGate(cpflags.NewOpenCatalog(), p.flags, p.refusals, p.store, p.clock)
	return gate.Call(t.Context(), clicks.Payer{Account: alice.String()}, alice, country) //nolint:wrapcheck // the test reads the gate's own error.
}

func TestARallyGoesAnywhereForTheCallersFlag(t *testing.T) {
	require.NoError(t, newGateParts().call(t, "fr"))
}

func TestARallyIsForNoFlagThatIsNoCountry(t *testing.T) {
	require.ErrorIs(t, newGateParts().call(t, "zz"), clicks.ErrUnknownCountry)
}

func TestARallyIsForTheFlagTheCallerPlaysFor(t *testing.T) {
	parts := newGateParts()
	parts.flags = flags{err: allegiances.ErrOtherFlag}

	require.ErrorIs(t, parts.call(t, "de"), allegiances.ErrOtherFlag)

	parts.flags = flags{err: fmt.Errorf("%w: postgres down", allegiances.ErrUnread)}
	require.NoError(t, parts.call(t, "de"), "a flag that cannot be read refuses nothing, as for a click")
}

func TestABannedCallerCallsNoRally(t *testing.T) {
	parts := newGateParts()
	parts.refusals = refusals{refusal: clicks.NewRefusal(calledAt.Add(time.Hour)), refused: true}

	var refusal clicks.Refusal
	require.ErrorAs(t, parts.call(t, "fr"), &refusal)

	parts.refusals = refusals{err: errors.New("postgres down")}
	require.NoError(t, parts.call(t, "fr"), "refusals that cannot be read refuse nothing")
}

func TestARallyOfTheCallerStillOnRefusesTheNext(t *testing.T) {
	parts := newGateParts()
	keep(t, parts.store, rallyOf(alice, "fr", 10, calledAt))

	require.ErrorIs(t, parts.call(t, "fr"), rallies.ErrRallyOn)

	parts.clock.Advance(2 * time.Minute)
	require.NoError(t, parts.call(t, "fr"), "once it ended")

	parts.store.FailWith(errors.New("postgres down"))
	assert.NoError(t, parts.call(t, "fr"), "a last that cannot be read refuses nothing: the call checks again")
}
