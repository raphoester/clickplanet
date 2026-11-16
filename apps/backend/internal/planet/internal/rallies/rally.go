// Package rallies is a rally's way from the hand to the globe, and what a click inside one costs: a ring on the globe
// that a player calls for its flag, inside which a click of that flag takes less from the bank until it ends.
package rallies

import (
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
)

var (
	// ErrRallyOn is a call while the account's last rally is still on: a player has one at a time.
	ErrRallyOn = errors.New("a rally the caller called is still on")
	// ErrNoPoint is a call aimed at no point of the globe: a vector of no length, or not a number.
	ErrNoPoint = errors.New("a rally names no point of the globe")
)

type RallyID uuid.UUID

func (id RallyID) String() string { return uuid.UUID(id).String() }

type IDProvider interface {
	NewID() (RallyID, error)
}

type Ground interface {
	Position(id uint32) (clicks.Vec3, bool)
}

// Rally is a ring a player called on the globe for its flag, from calledAt to endsAt, around a point: not a tile, since
// the ring is what it covers.
type Rally struct {
	id       RallyID
	account  cpsession.AccountID
	flag     string
	centre   clicks.Vec3
	radius   float64
	calledAt time.Time
	endsAt   time.Time
}

// On says whether it is still on at that time.
func (r Rally) On(at time.Time) bool {
	return at.Before(r.endsAt)
}

// Called is the rally next, the account's last from now on: refused while this one is still on as next is called.
func (r Rally) Called(next Rally) (Rally, error) {
	if r.On(next.calledAt) {
		return r, fmt.Errorf("%w until %s", ErrRallyOn, r.endsAt.Format(time.RFC3339))
	}

	return next, nil
}

// Covers says whether the tile is inside the ring: its centre within the radius of the ring's, in arc on the globe.
func (r Rally) Covers(tile uint32, ground Ground) bool {
	position, ok := ground.Position(tile)
	if !ok {
		return false
	}

	a, b := r.centre.Unit(), position.Unit()
	return a.X*b.X+a.Y*b.Y+a.Z*b.Z >= math.Cos(r.radius)
}

// PointOf is a point of the globe for a vector of any length: on the unit globe.
func PointOf(vector clicks.Vec3) (clicks.Vec3, error) {
	length := math.Sqrt(vector.X*vector.X + vector.Y*vector.Y + vector.Z*vector.Z)
	if length == 0 || math.IsNaN(length) || math.IsInf(length, 0) {
		return clicks.Vec3{}, fmt.Errorf("%w: %+v", ErrNoPoint, vector)
	}

	return vector.Unit(), nil
}

// Snapshot is a rally as postgres keeps it and the wire tells it.
type Snapshot struct {
	ID       RallyID
	Account  cpsession.AccountID
	Flag     string
	Centre   clicks.Vec3
	Radius   float64
	CalledAt time.Time
	EndsAt   time.Time
}

func (r Rally) TakeSnapshot() Snapshot {
	return Snapshot{
		ID: r.id, Account: r.account, Flag: r.flag, Centre: r.centre, Radius: r.radius, CalledAt: r.calledAt, EndsAt: r.endsAt,
	}
}

func (s Snapshot) Restore() Rally {
	return Rally{
		id: s.ID, account: s.Account, flag: s.Flag, centre: s.Centre, radius: s.Radius, calledAt: s.CalledAt, endsAt: s.EndsAt,
	}
}
