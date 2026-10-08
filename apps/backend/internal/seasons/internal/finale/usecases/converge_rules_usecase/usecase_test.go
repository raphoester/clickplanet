package converge_rules_usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/calendar"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/finale"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/finale/usecases/converge_rules_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var (
	finaleStarts = time.Date(2026, 10, 31, 21, 0, 0, 0, time.UTC)
	seasonEnds   = time.Date(2026, 10, 31, 23, 0, 0, 0, time.UTC)
)

type recordingPlanet struct {
	set  []finale.Switches
	fail error
}

func (r *recordingPlanet) Set(_ context.Context, switches finale.Switches) error {
	if r.fail != nil {
		return r.fail
	}
	r.set = append(r.set, switches)
	return nil
}

func (r *recordingPlanet) last() finale.Switches { return r.set[len(r.set)-1] }

func seasonZero() calendar.Calendar {
	return calendar.New(calendar.Config{List: []calendar.Entry{{Number: 0, EndsAt: seasonEnds, Finale: 2 * time.Hour}}})
}

func converge(clock cptime.Clock, planet *recordingPlanet) *converge_rules_usecase.UseCase {
	return converge_rules_usecase.New(seasonZero(), finale.NewRules(finale.Config{}), clock, planet)
}

func TestABootInTheMiddleOfTheFinaleSetsTheFinalesRulesAtOnce(t *testing.T) {
	planet := &recordingPlanet{}

	require.NoError(t, converge(cptime.NewFixedClock(finaleStarts.Add(time.Hour)), planet).Execute(t.Context()))

	require.Len(t, planet.set, 1)
	assert.InDelta(t, 3.0, planet.last().RefillMultiplier(), 1e-9)
	_, _, giving := planet.last().Gift()
	assert.True(t, giving)
}

func TestRulesPlanetConfirmedAreNotSetAgain(t *testing.T) {
	planet := &recordingPlanet{}
	clock := cptime.NewFixedClock(finaleStarts.Add(time.Hour))
	useCase := converge(clock, planet)

	for range 5 {
		clock.Advance(time.Second)
		require.NoError(t, useCase.Execute(t.Context()))
	}

	assert.Len(t, planet.set, 1)
}

func TestEachChangeOfPhaseIsSetOnce(t *testing.T) {
	planet := &recordingPlanet{}
	clock := cptime.NewFixedClock(finaleStarts.Add(-2 * time.Second))
	useCase := converge(clock, planet)

	for range 4 * 60 * 60 {
		require.NoError(t, useCase.Execute(t.Context()))
		clock.Advance(time.Second)
	}

	require.Len(t, planet.set, 3, "plain, the finale, then frozen")
	assert.True(t, planet.set[0].Equal(finale.Switches{}))
	assert.InDelta(t, 3.0, planet.set[1].RefillMultiplier(), 1e-9)
	assert.True(t, planet.set[2].Frozen())
}

func TestARefusedSetIsTriedAgainUntilPlanetTakesIt(t *testing.T) {
	planet := &recordingPlanet{fail: errors.New("connection refused")}
	clock := cptime.NewFixedClock(seasonEnds.Add(time.Minute))
	useCase := converge(clock, planet)

	require.Error(t, useCase.Execute(t.Context()))
	require.Error(t, useCase.Execute(t.Context()))

	planet.fail = nil
	require.NoError(t, useCase.Execute(t.Context()))
	require.Len(t, planet.set, 1)
	assert.True(t, planet.last().Frozen())

	require.NoError(t, useCase.Execute(t.Context()))
	assert.Len(t, planet.set, 1)
}
