package days

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
)

type Calendar interface {
	DayAt(ctx context.Context, at time.Time) (Day, error)
}

// The facts of a batch that run in one day and share a key, an account or a flag, in the order they came.
type Group[K comparable, F any] struct {
	Day   Day
	Key   K
	Facts []F
}

// GroupsOf is a batch's facts by the day each runs in and by key, in the order each group's first fact came; a fact no
// day runs at is left out. Each day is read once, since a batch's facts mostly run in one.
func GroupsOf[K comparable, F any](
	ctx context.Context, calendar Calendar, facts []F, of func(F) (time.Time, K),
) ([]Group[K, F], error) {
	type place struct {
		day DayID
		key K
	}
	var (
		groups []Group[K, F]
		placed = map[place]int{}
		dated  = dates{calendar: calendar, none: cpcolls.NewSet[int64]()}
	)
	for _, fact := range facts {
		at, key := of(fact)
		day, runs, err := dated.dayAt(ctx, at)
		if err != nil {
			return nil, err
		}
		if !runs {
			continue
		}
		where := place{day: day.id, key: key}
		index, known := placed[where]
		if !known {
			index = len(groups)
			placed[where] = index
			groups = append(groups, Group[K, F]{Day: day, Key: key})
		}
		groups[index].Facts = append(groups[index].Facts, fact)
	}
	return groups, nil
}

// The days read for one batch, and the times none runs at. Not kept past it: a day told again may end sooner.
type dates struct {
	calendar Calendar
	read     []Day
	none     *cpcolls.Set[int64]
}

func (d *dates) dayAt(ctx context.Context, at time.Time) (Day, bool, error) {
	for _, day := range d.read {
		if day.Runs(at) {
			return day, true, nil
		}
	}
	if d.none.Contains(at.UnixNano()) {
		return Day{}, false, nil
	}
	day, err := d.calendar.DayAt(ctx, at)
	if errors.Is(err, ErrNoDay) {
		d.none.Add(at.UnixNano())
		return Day{}, false, nil
	}
	if err != nil {
		return Day{}, false, fmt.Errorf("failed to read the day at %s: %w", at.Format(time.RFC3339Nano), err)
	}
	d.read = append(d.read, day)
	return day, true, nil
}
