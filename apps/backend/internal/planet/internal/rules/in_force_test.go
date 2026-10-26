package rules_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpratelimit"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var epoch = time.Date(2026, 10, 31, 21, 0, 0, 0, time.UTC)

func setOf(t *testing.T, snapshot rules.Snapshot) rules.Set {
	t.Helper()

	set, err := snapshot.Restore()
	require.NoError(t, err)
	return set
}

func TestNothingPlaysBeforeTheFirstSet(t *testing.T) {
	inForce := rules.NewInForce(cptime.NewFixedClock(epoch))

	assert.Zero(t, inForce.Tempo(), "the antibot reads 0 as the plain tempo")
	assert.Zero(t, inForce.Policy().Burst, "a bank of 0 lets no click through")
	assert.Zero(t, inForce.Bonuses().Boxes.MaxChargesPerHour, "a cap of 0 offers no box")
	assert.Zero(t, inForce.ShieldsPerTile())
	assert.Empty(t, inForce.Toll())
}

func TestEveryReaderReadsTheSetEnforced(t *testing.T) {
	clock := cptime.NewFixedClock(epoch)
	inForce := rules.NewInForce(clock)
	rush := rushSnapshot()
	rush.Bonuses.Shield.PerTile = 20

	inForce.Enforce(setOf(t, rush))

	assert.InDelta(t, 3.0, inForce.Tempo(), 1e-9)
	assert.Equal(t, cpratelimit.Policy{PerSecond: rush.Clicks.PerSecond, Burst: rush.Clicks.Burst, Tempo: 3, Since: epoch},
		inForce.Policy())
	assert.Equal(t, rush.Clicks, inForce.Throttle())
	assert.Equal(t, rush.Toll, inForce.Toll())
	assert.Equal(t, rush.Bonuses, inForce.Bonuses())
	assert.Equal(t, 20, inForce.ShieldsPerTile())
}

func TestAPolicySaysWhenItTookOver(t *testing.T) {
	clock := cptime.NewFixedClock(epoch)
	inForce := rules.NewInForce(clock)
	inForce.Enforce(setOf(t, rules.TestSnapshot(rules.Default)))

	clock.Advance(time.Minute)
	inForce.Enforce(setOf(t, rushSnapshot()))

	assert.Equal(t, epoch.Add(time.Minute), inForce.Policy().Since)
}

func TestASetEnforcedReplacesTheOneBefore(t *testing.T) {
	inForce := rules.NewInForce(cptime.NewFixedClock(epoch))
	inForce.Enforce(setOf(t, rushSnapshot()))

	inForce.Enforce(setOf(t, rules.TestSnapshot(rules.Default)))

	assert.InDelta(t, 1.0, inForce.Tempo(), 1e-9)
	assert.Equal(t, bonuses.TestRules().Boxes, inForce.Bonuses().Boxes)
}

func TestEveryWatcherHearsOfEachSetEnforced(t *testing.T) {
	inForce := rules.NewInForce(cptime.NewFixedClock(epoch))
	first, leaveFirst := inForce.Watch()
	defer leaveFirst()
	second, leaveSecond := inForce.Watch()
	defer leaveSecond()

	inForce.Enforce(setOf(t, rushSnapshot()))

	assert.Equal(t, rushSnapshot(), (<-first).TakeSnapshot())
	assert.Equal(t, rushSnapshot(), (<-second).TakeSnapshot())
}

func TestAWatcherThatFellBehindHearsOnlyTheLatest(t *testing.T) {
	inForce := rules.NewInForce(cptime.NewFixedClock(epoch))
	watcher, leave := inForce.Watch()
	defer leave()

	inForce.Enforce(setOf(t, rushSnapshot()))
	inForce.Enforce(setOf(t, rules.TestSnapshot(rules.Default)))

	assert.Equal(t, rules.TestSnapshot(rules.Default), (<-watcher).TakeSnapshot())
	select {
	case set := <-watcher:
		require.Failf(t, "a second set waited", "%+v", set.TakeSnapshot())
	default:
	}
}

func TestAWatcherThatLeftHearsNothing(t *testing.T) {
	inForce := rules.NewInForce(cptime.NewFixedClock(epoch))
	watcher, leave := inForce.Watch()
	leave()

	inForce.Enforce(setOf(t, rushSnapshot()))

	select {
	case set := <-watcher:
		require.Failf(t, "a watcher that left heard a set", "%+v", set.TakeSnapshot())
	default:
	}
}
