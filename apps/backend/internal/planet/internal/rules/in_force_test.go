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

var rush = rules.Set{
	Name:  "rush",
	Tempo: 3,
	Boxes: bonuses.Pace{MinInterval: time.Minute, MaxInterval: 2 * time.Minute, MaxChargesPerHour: 30},
}

func TestTheRulesInForceStartAsGivenAndTellNobody(t *testing.T) {
	tempo, schedule := &recordedTempo{}, &recordedSchedule{}
	inForce := rules.NewInForce(defaults, tempo, schedule)

	assert.InDelta(t, 1.0, inForce.Tempo(), 1e-9)
	assert.False(t, inForce.Frozen())
	assert.Empty(t, tempo.set)
	assert.Empty(t, schedule.set)
	assert.Empty(t, schedule.frozen)
}

func TestASetEnforcedSetsTheTempoAndTheBoxes(t *testing.T) {
	tempo, schedule := &recordedTempo{}, &recordedSchedule{}
	inForce := rules.NewInForce(defaults, tempo, schedule)

	inForce.Enforce(rush)

	assert.InDelta(t, 3.0, inForce.Tempo(), 1e-9)
	assert.Equal(t, []float64{3}, tempo.set)
	assert.Equal(t, []bonuses.Pace{rush.Boxes}, schedule.set)
	assert.Equal(t, []bool{false}, schedule.frozen)
}

func TestAFrozenSetFreezesTheMapAndTheBoxesUntilTheNextSet(t *testing.T) {
	schedule := &recordedSchedule{}
	inForce := rules.NewInForce(defaults, &recordedTempo{}, schedule)

	inForce.Enforce(rules.Set{Name: "frozen", Tempo: 1, Boxes: defaults.Boxes, Frozen: true})
	assert.True(t, inForce.Frozen())

	inForce.Enforce(defaults)
	assert.False(t, inForce.Frozen())

	assert.Equal(t, []bool{true, false}, schedule.frozen)
}

func TestEveryWatcherHearsOfEachSetEnforced(t *testing.T) {
	inForce := rules.NewInForce(defaults, &recordedTempo{}, &recordedSchedule{})
	first, leaveFirst := inForce.Watch()
	defer leaveFirst()
	second, leaveSecond := inForce.Watch()
	defer leaveSecond()

	inForce.Enforce(rush)

	assert.Equal(t, rush, <-first)
	assert.Equal(t, rush, <-second)
}

func TestAWatcherThatFellBehindHearsOnlyTheLatest(t *testing.T) {
	inForce := rules.NewInForce(defaults, &recordedTempo{}, &recordedSchedule{})
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
	inForce := rules.NewInForce(defaults, &recordedTempo{}, &recordedSchedule{})
	watcher, leave := inForce.Watch()
	leave()

	inForce.Enforce(rush)

	select {
	case set := <-watcher:
		require.Failf(t, "a watcher that left heard a set", "%+v", set)
	default:
	}
}
