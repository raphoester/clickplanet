package ledger

import (
	"errors"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
)

var ErrInvalidSpan = errors.New("a span of activity is more than nothing and at most 72 hours")

var activitySteps = []struct {
	upTo time.Duration
	step time.Duration
}{
	{upTo: 6 * time.Hour, step: 5 * time.Minute},
	{upTo: 24 * time.Hour, step: 15 * time.Minute},
	{upTo: defaultRetention, step: time.Hour},
}

// The slots start on the step, so the last one is still filling at until.
func NewActivity(until time.Time, span time.Duration) (*Activity, error) {
	if span <= 0 {
		return nil, ErrInvalidSpan
	}

	for _, bound := range activitySteps {
		if span > bound.upTo {
			continue
		}

		count := int((span + bound.step - 1) / bound.step)
		slots := make([]Slot, count)
		takers := make([]*cpcolls.Set[Caller], count)
		since := until.Truncate(bound.step).Add(-time.Duration(count-1) * bound.step)
		for i := range slots {
			slots[i].Start = since.Add(time.Duration(i) * bound.step)
			takers[i] = cpcolls.NewSet[Caller]()
		}

		return &Activity{
			since: since, until: until, step: bound.step, slots: slots, takers: takers,
			players: cpcolls.NewSet[Caller](), changes: make(map[uint32]int), moves: make(map[string]Moves),
		}, nil
	}

	return nil, ErrInvalidSpan
}

type Activity struct {
	since   time.Time
	until   time.Time
	step    time.Duration
	slots   []Slot
	takers  []*cpcolls.Set[Caller]
	players *cpcolls.Set[Caller]
	changes map[uint32]int
	moves   map[string]Moves
}

type Slot struct {
	Start   time.Time
	Clicks  int
	Tiles   int
	Players int
}

type Moves struct {
	Won  int
	Lost int
}

type Summary struct {
	Since   time.Time
	Until   time.Time
	Step    time.Duration
	Slots   []Slot
	Players int
	Changes map[uint32]int
	Moves   map[string]Moves
}

func (a *Activity) See(event Event) {
	event.Show(func(scene Scene) {
		if scene.Change == nil || !scene.Change.Clicked {
			return
		}
		if i, ok := a.slotOf(scene.At); ok {
			a.slots[i].Clicks++
		}
	})

	event.Replay(func(taking Taking) {
		i, ok := a.slotOf(taking.At)
		if !ok {
			return
		}

		a.slots[i].Tiles++
		a.takers[i].Add(taking.Taker())
		a.players.Add(taking.Taker())
		a.changes[taking.Tile]++

		if !taking.Cleared() {
			won := a.moves[taking.Country]
			won.Won++
			a.moves[taking.Country] = won
		}
		if taking.Previous != "" {
			lost := a.moves[taking.Previous]
			lost.Lost++
			a.moves[taking.Previous] = lost
		}
	})
}

func (a *Activity) slotOf(at time.Time) (int, bool) {
	if at.Before(a.since) || !at.Before(a.until) {
		return 0, false
	}

	return int(at.Sub(a.since) / a.step), true
}

func (a *Activity) Summary() Summary {
	slots := make([]Slot, len(a.slots))
	for i, slot := range a.slots {
		slot.Players = a.takers[i].Len()
		slots[i] = slot
	}

	changes := make(map[uint32]int, len(a.changes))
	for tile, times := range a.changes {
		changes[tile] = times
	}
	moves := make(map[string]Moves, len(a.moves))
	for country, moved := range a.moves {
		moves[country] = moved
	}

	return Summary{
		Since: a.since, Until: a.until, Step: a.step, Slots: slots, Players: a.players.Len(),
		Changes: changes, Moves: moves,
	}
}
