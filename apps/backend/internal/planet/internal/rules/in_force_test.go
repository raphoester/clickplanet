package rules_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules"
)

type recordedTempo struct{ set []float64 }

func (r *recordedTempo) SetTempo(tempo float64) { r.set = append(r.set, tempo) }

type recordedSchedule struct {
	set    []bonuses.Pace
	frozen []bool
}

func (r *recordedSchedule) SetPace(pace bonuses.Pace) { r.set = append(r.set, pace) }

func (r *recordedSchedule) SetFrozen(frozen bool) { r.frozen = append(r.frozen, frozen) }

type recordedGuard struct{ set []int }

func (r *recordedGuard) SetCrowd(members int) { r.set = append(r.set, members) }

var rush = rules.Set{
	Name:    "rush",
	Tempo:   3,
	Boxes:   bonuses.Pace{MinInterval: time.Minute, MaxInterval: 2 * time.Minute, MaxChargesPerHour: 30},
	AntiBot: rules.AntiBot{Crowd: 50},
}

func TestTheRulesInForceStartAsGivenAndTellNobody(t *testing.T) {
	tempo, schedule, guard := &recordedTempo{}, &recordedSchedule{}, &recordedGuard{}
	inForce := rules.NewInForce(defaults, tempo, schedule, guard)

	assert.InDelta(t, 1.0, inForce.Tempo(), 1e-9)
	assert.False(t, inForce.Frozen())
	assert.Empty(t, tempo.set)
	assert.Empty(t, schedule.set)
	assert.Empty(t, schedule.frozen)
	assert.Empty(t, guard.set)
}

func TestASetEnforcedSetsTheTempoTheBoxesAndTheCrowd(t *testing.T) {
	tempo, schedule, guard := &recordedTempo{}, &recordedSchedule{}, &recordedGuard{}
	inForce := rules.NewInForce(defaults, tempo, schedule, guard)

	inForce.Enforce(rush)

	assert.InDelta(t, 3.0, inForce.Tempo(), 1e-9)
	assert.Equal(t, []float64{3}, tempo.set)
	assert.Equal(t, []bonuses.Pace{rush.Boxes}, schedule.set)
	assert.Equal(t, []int{50}, guard.set)
}

func TestAFrozenSetFreezesTheMapAndTheBoxesUntilTheNextSet(t *testing.T) {
	schedule := &recordedSchedule{}
	inForce := rules.NewInForce(defaults, &recordedTempo{}, schedule, &recordedGuard{})

	inForce.Enforce(rules.Set{Name: "frozen", Tempo: 1, Boxes: defaults.Boxes, Frozen: true})
	assert.True(t, inForce.Frozen())

	inForce.Enforce(defaults)
	assert.False(t, inForce.Frozen())

	assert.Equal(t, []bool{true, false}, schedule.frozen)
}

func TestEveryWatcherHearsOfEachSetEnforced(t *testing.T) {
	inForce := rules.NewInForce(defaults, &recordedTempo{}, &recordedSchedule{}, &recordedGuard{})
	first, leaveFirst := inForce.Watch()
	defer leaveFirst()
	second, leaveSecond := inForce.Watch()
	defer leaveSecond()

	inForce.Enforce(rush)

	assert.Equal(t, rush, <-first)
	assert.Equal(t, rush, <-second)
}

func TestAWatcherThatFellBehindHearsOnlyTheLatest(t *testing.T) {
	inForce := rules.NewInForce(defaults, &recordedTempo{}, &recordedSchedule{}, &recordedGuard{})
	watcher, leave := inForce.Watch()
	defer leave()

	inForce.Enforce(rush)
	inForce.Enforce(defaults)

	assert.Equal(t, defaults, <-watcher)
	select {
	case set := <-watcher:
		require.Failf(t, "a second set waited", "%+v", set)
	default:
	}
}

func TestAWatcherThatLeftHearsNothing(t *testing.T) {
	inForce := rules.NewInForce(defaults, &recordedTempo{}, &recordedSchedule{}, &recordedGuard{})
	watcher, leave := inForce.Watch()
	leave()

	inForce.Enforce(rush)

	select {
	case set := <-watcher:
		require.Failf(t, "a watcher that left heard a set", "%+v", set)
	default:
	}
}
