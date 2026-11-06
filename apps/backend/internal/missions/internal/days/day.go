package days

import (
	"context"
	"errors"
	"fmt"
	"time"
)

var (
	ErrNoDay      = errors.New("no day runs then")
	ErrInvalidDay = errors.New("a day has an id and a number, and ends after it starts")
)

// DayID is the id the gamemaster gave the day's round: missions only repeats it.
type DayID string

type DaySnapshot struct {
	ID DayID
	// The day's place among its season's days, from 1, as the gamemaster numbers them.
	Season, Number uint32
	StartsAt       time.Time
	EndsAt         time.Time
}

// A day as the gamemaster told it: missions are drawn, counted and paid by day.
type Day struct {
	id             DayID
	season, number uint32
	startsAt       time.Time
	endsAt         time.Time
}

func DayOf(snapshot DaySnapshot) (Day, error) {
	if snapshot.ID == "" || snapshot.Number == 0 || !snapshot.EndsAt.After(snapshot.StartsAt) {
		return Day{}, fmt.Errorf("%w: %q, day %d of season %d, from %s to %s", ErrInvalidDay, snapshot.ID, snapshot.Number,
			snapshot.Season, snapshot.StartsAt, snapshot.EndsAt)
	}
	return Day{
		id: snapshot.ID, season: snapshot.Season, number: snapshot.Number, startsAt: snapshot.StartsAt.UTC(),
		endsAt: snapshot.EndsAt.UTC(),
	}, nil
}

func (d Day) Snapshot() DaySnapshot {
	return DaySnapshot{ID: d.id, Season: d.season, Number: d.number, StartsAt: d.startsAt, EndsAt: d.endsAt}
}

func (d Day) Runs(at time.Time) bool {
	return !at.Before(d.startsAt) && at.Before(d.endsAt)
}

// Follows is true when the day is the next one of the other's season. A battle between them does not change it: it
// takes its time from a day, so the day before ends before the day starts.
func (d Day) Follows(other Day) bool {
	return d.season == other.season && d.number == other.number+1
}

// The day's key in the stores.
func (d Day) String() string { return string(d.id) }

type Store interface {
	// DayAt is the day told that runs at a time: ErrNoDay when none does.
	DayAt(ctx context.Context, at time.Time) (Day, error)
	// DayBefore is the day told before a day in its season, cut short by a battle or not: ErrNoDay on a season's first
	// day.
	DayBefore(ctx context.Context, day Day) (Day, error)
	// KeepDay keeps a day told, or the new times of a day told again.
	KeepDay(ctx context.Context, day Day) error
}
