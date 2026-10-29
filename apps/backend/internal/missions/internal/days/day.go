package days

import (
	"context"
	"errors"
	"fmt"
	"time"
)

var (
	ErrNoDay      = errors.New("no day runs then")
	ErrInvalidDay = errors.New("a day has an id and ends after it starts")
)

// DayID is the id the gamemaster gave the day's round: missions only repeats it.
type DayID string

type DaySnapshot struct {
	ID       DayID
	StartsAt time.Time
	EndsAt   time.Time
}

// A day as the gamemaster told it: missions are drawn, counted and paid by day.
type Day struct {
	id       DayID
	startsAt time.Time
	endsAt   time.Time
}

func DayOf(snapshot DaySnapshot) (Day, error) {
	if snapshot.ID == "" || !snapshot.EndsAt.After(snapshot.StartsAt) {
		return Day{}, fmt.Errorf("%w: %q from %s to %s", ErrInvalidDay, snapshot.ID, snapshot.StartsAt, snapshot.EndsAt)
	}
	return Day{id: snapshot.ID, startsAt: snapshot.StartsAt.UTC(), endsAt: snapshot.EndsAt.UTC()}, nil
}

func (d Day) Snapshot() DaySnapshot {
	return DaySnapshot{ID: d.id, StartsAt: d.startsAt, EndsAt: d.endsAt}
}

func (d Day) Runs(at time.Time) bool {
	return !at.Before(d.startsAt) && at.Before(d.endsAt)
}

// Follows is true when the day starts as the other ends: the gamemaster chains a season's days.
func (d Day) Follows(other Day) bool {
	return d.startsAt.Equal(other.endsAt)
}

// The day's key in the stores.
func (d Day) String() string { return string(d.id) }

type Store interface {
	// DayAt is the day told that runs at a time: ErrNoDay when none does.
	DayAt(ctx context.Context, at time.Time) (Day, error)
	// DayBefore is the day told that ended as a day started: ErrNoDay on a season's first day.
	DayBefore(ctx context.Context, day Day) (Day, error)
	// KeepDay keeps a day told, or the new times of a day told again.
	KeepDay(ctx context.Context, day Day) error
}
