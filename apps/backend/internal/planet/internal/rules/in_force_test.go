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

type recordedSchedule struct{ set []bonuses.Pace }

func (r *recordedSchedule) SetPace(pace bonuses.Pace) { r.set = append(r.set, pace) }

type recordedGuard struct{ set []int }

func (r *recordedGuard) SetCrowd(members int) { r.set = append(r.set, members) }

func setOf(t *testing.T, snapshot rules.Snapshot) rules.Set {
	t.Helper()

	set, err := snapshot.Restore()
	require.NoError(t, err)
	return set
}

var defaultSnapshot = rules.Snapshot{
	Name:  rules.Default,
	Tempo: 1,
	Boxes: bonuses.Pace{MinInterval: 4 * time.Minute, MaxInterval: 8 * time.Minute, MaxChargesPerHour: 12},
	Crowd: 6,
}

func TestNothingIsInForceBeforeTheFirstSet(t *testing.T) {
	tempo, schedule, guard := &recordedTempo{}, &recordedSchedule{}, &recordedGuard{}
	inForce := rules.NewInForce(tempo, schedule, guard)

	assert.Zero(t, inForce.Tempo(), "the antibot reads 0 as the plain tempo")
	assert.Empty(t, tempo.set)
	assert.Empty(t, schedule.set)
	assert.Empty(t, guard.set)
}

func TestASetEnforcedSetsTheTempoTheBoxesAndTheCrowd(t *testing.T) {
	tempo, schedule, guard := &recordedTempo{}, &recordedSchedule{}, &recordedGuard{}
	inForce := rules.NewInForce(tempo, schedule, guard)

	inForce.Enforce(setOf(t, rushSnapshot))

	assert.InDelta(t, 3.0, inForce.Tempo(), 1e-9)
	assert.Equal(t, []float64{3}, tempo.set)
	assert.Equal(t, []bonuses.Pace{rushSnapshot.Boxes}, schedule.set)
	assert.Equal(t, []int{50}, guard.set)
}

func TestEveryWatcherHearsOfEachSetEnforced(t *testing.T) {
	inForce := rules.NewInForce(&recordedTempo{}, &recordedSchedule{}, &recordedGuard{})
	first, leaveFirst := inForce.Watch()
	defer leaveFirst()
	second, leaveSecond := inForce.Watch()
	defer leaveSecond()

	inForce.Enforce(setOf(t, rushSnapshot))

	assert.Equal(t, rushSnapshot, (<-first).TakeSnapshot())
	assert.Equal(t, rushSnapshot, (<-second).TakeSnapshot())
}

func TestAWatcherThatFellBehindHearsOnlyTheLatest(t *testing.T) {
	inForce := rules.NewInForce(&recordedTempo{}, &recordedSchedule{}, &recordedGuard{})
	watcher, leave := inForce.Watch()
	defer leave()

	inForce.Enforce(setOf(t, rushSnapshot))
	inForce.Enforce(setOf(t, defaultSnapshot))

	assert.Equal(t, defaultSnapshot, (<-watcher).TakeSnapshot())
	select {
	case set := <-watcher:
		require.Failf(t, "a second set waited", "%+v", set.TakeSnapshot())
	default:
	}
}

func TestAWatcherThatLeftHearsNothing(t *testing.T) {
	inForce := rules.NewInForce(&recordedTempo{}, &recordedSchedule{}, &recordedGuard{})
	watcher, leave := inForce.Watch()
	leave()

	inForce.Enforce(setOf(t, rushSnapshot))

	select {
	case set := <-watcher:
		require.Failf(t, "a watcher that left heard a set", "%+v", set.TakeSnapshot())
	default:
	}
}
