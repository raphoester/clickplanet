package call_rally_usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/charges"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rallies"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rallies/usecases/call_rally_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

const alice charges.Holder = "01926c6e-7a4b-7c3d-8e9f-0a1b2c3d4e5f"

var (
	aliceID  = cpsession.AccountID{0x01, 0x92, 0x6c, 0x6e, 0x7a, 0x4b, 0x7c, 0x3d, 0x8e, 0x9f, 0x0a, 0x1b, 0x2c, 0x3d, 0x4e, 0x5f}
	calledAt = time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)
	north    = clicks.Vec3{Y: 1}
)

type gate struct{ err error }

func (g gate) Call(context.Context, clicks.Payer, cpsession.AccountID, string) error {
	return g.err
}

type parts struct {
	hands *charges.MemoryStore
	calls *rallies.MemoryStore
	told  *rallies.RecordedCalls
	clock *cptime.FixedClock
}

func setup(refused error, held charges.Held) (*call_rally_usecase.UseCase, parts) {
	p := parts{
		hands: charges.NewMemoryStore(), calls: rallies.NewMemoryStore(), told: &rallies.RecordedCalls{},
		clock: cptime.NewFixedClock(calledAt),
	}
	p.hands.Keep(alice, held)
	muster := rallies.NewMuster(charges.TestRules, 0.004, &rallies.SequentialIDs{}, p.clock)
	return call_rally_usecase.New(gate{err: refused}, charges.NewCharges(p.hands, charges.TestRules), muster, p.calls, p.told), p
}

func calling(t *testing.T) context.Context {
	t.Helper()
	return cpctx.AddAccountToContext(cpctx.AddIPToContext(t.Context(), "203.0.113.7"), string(alice))
}

func TestARallyLeavesTheHandIsKeptAndIsToldOnce(t *testing.T) {
	useCase, p := setup(nil, charges.Held{Rallies: 2, Shields: 2})

	rally, err := useCase.Execute(calling(t), call_rally_usecase.In{Point: north, CountryID: "fr"})
	require.NoError(t, err)

	kept := rally.TakeSnapshot()
	assert.Equal(t, charges.Held{Rallies: 1, Shields: 2}, p.hands.Kept(alice))
	assert.Equal(t, aliceID, kept.Account)
	assert.Equal(t, "fr", kept.Flag)
	assert.Equal(t, north, kept.Centre)
	assert.Equal(t, calledAt.Add(3*time.Minute), kept.EndsAt)
	last, err := p.calls.Last(t.Context(), aliceID)
	require.NoError(t, err)
	assert.Equal(t, kept, last.TakeSnapshot())
	assert.Equal(t, []rallies.Snapshot{kept}, p.told.Told())
}

func TestARallyTheGateRefusesStaysInTheHand(t *testing.T) {
	useCase, p := setup(rallies.ErrRallyOn, charges.Held{Rallies: 1})

	_, err := useCase.Execute(calling(t), call_rally_usecase.In{Point: north, CountryID: "fr"})

	require.ErrorIs(t, err, rallies.ErrRallyOn)
	assert.Equal(t, charges.Held{Rallies: 1}, p.hands.Kept(alice))
	assert.Empty(t, p.told.Told())
}

func TestARallyStillOnUnderTheLockGivesTheSpentOneBack(t *testing.T) {
	useCase, p := setup(nil, charges.Held{Rallies: 2})
	_, err := useCase.Execute(calling(t), call_rally_usecase.In{Point: north, CountryID: "fr"})
	require.NoError(t, err)

	p.clock.Advance(time.Minute)
	_, err = useCase.Execute(calling(t), call_rally_usecase.In{Point: clicks.Vec3{X: 1}, CountryID: "fr"})

	require.ErrorIs(t, err, rallies.ErrRallyOn, "the gate let it through, as a second tab's would")
	assert.Equal(t, charges.Held{Rallies: 1}, p.hands.Kept(alice))
	assert.Len(t, p.told.Told(), 1)
}

func TestAnotherRallyOnceTheLastEnded(t *testing.T) {
	useCase, p := setup(nil, charges.Held{Rallies: 2})
	_, err := useCase.Execute(calling(t), call_rally_usecase.In{Point: north, CountryID: "fr"})
	require.NoError(t, err)

	p.clock.Advance(3 * time.Minute)
	_, err = useCase.Execute(calling(t), call_rally_usecase.In{Point: clicks.Vec3{X: 1}, CountryID: "fr"})

	require.NoError(t, err)
	assert.Equal(t, charges.Held{}, p.hands.Kept(alice))
	assert.Len(t, p.told.Told(), 2)
}

func TestNoRallyHeldIsNoRally(t *testing.T) {
	useCase, p := setup(nil, charges.Held{Bomb: true})

	_, err := useCase.Execute(calling(t), call_rally_usecase.In{Point: north, CountryID: "fr"})

	require.ErrorIs(t, err, call_rally_usecase.ErrNoRally)
	assert.Empty(t, p.told.Told())
}

func TestACallerWithNoAccountHoldsNoRally(t *testing.T) {
	useCase, _ := setup(nil, charges.Held{Rallies: 1})

	_, err := useCase.Execute(cpctx.AddIPToContext(t.Context(), "203.0.113.7"), call_rally_usecase.In{Point: north, CountryID: "fr"})

	require.ErrorIs(t, err, call_rally_usecase.ErrNoRally)
}

func TestARallyThatCannotBeKeptComesBack(t *testing.T) {
	useCase, p := setup(nil, charges.Held{Rallies: 1})
	p.calls.FailWith(errors.New("postgres down"))

	_, err := useCase.Execute(calling(t), call_rally_usecase.In{Point: north, CountryID: "fr"})

	require.Error(t, err)
	assert.Equal(t, charges.Held{Rallies: 1}, p.hands.Kept(alice))
	assert.Empty(t, p.told.Told())
}

func TestARallyOnNoPointStaysInTheHand(t *testing.T) {
	useCase, p := setup(nil, charges.Held{Rallies: 1})

	_, err := useCase.Execute(calling(t), call_rally_usecase.In{CountryID: "fr"})

	require.ErrorIs(t, err, rallies.ErrNoPoint)
	assert.Equal(t, charges.Held{Rallies: 1}, p.hands.Kept(alice))
	assert.Empty(t, p.told.Told())
}

func TestARallyIsOnThePointOfTheGlobeUnderTheTap(t *testing.T) {
	useCase, _ := setup(nil, charges.Held{Rallies: 1})

	rally, err := useCase.Execute(calling(t), call_rally_usecase.In{Point: clicks.Vec3{Z: 2}, CountryID: "fr"})

	require.NoError(t, err)
	assert.Equal(t, clicks.Vec3{Z: 1}, rally.TakeSnapshot().Centre, "on the unit globe")
}
