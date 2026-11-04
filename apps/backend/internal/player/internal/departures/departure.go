package departures

import (
	"context"
	"errors"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
)

var ErrNoTime = errors.New("the decision says no time")

// An account its player asked to delete leaves: the other players see it as deleted until auth deletes it, or a
// sign-in brings it back.
type Departure struct {
	leaving   bool
	decidedAt time.Time
}

type Decision struct {
	leaving bool
	at      time.Time
}

func Leaves(at time.Time) Decision {
	return Decision{leaving: true, at: at}
}

func Returns(at time.Time) Decision {
	return Decision{at: at}
}

func (d Decision) TakeSnapshot() DecisionSnapshot {
	return DecisionSnapshot{Leaving: d.leaving, At: d.at}
}

type DecisionSnapshot struct {
	Leaving bool
	At      time.Time
}

func (d Decision) TimeError() error {
	if d.at.IsZero() {
		return ErrNoTime
	}
	return nil
}

// Only a later decision changes a departure: auth tells a request and a restore on two streams, which may come in
// either order, and either may come twice.
func (d Departure) Decided(decision Decision) Departure {
	if !decision.at.After(d.decidedAt) {
		return d
	}
	return Departure{leaving: decision.leaving, decidedAt: decision.at}
}

func (d Departure) Leaving() bool {
	return d.leaving
}

func (d Departure) TakeSnapshot() Snapshot {
	return Snapshot{Leaving: d.leaving, DecidedAt: d.decidedAt}
}

type Snapshot struct {
	Leaving   bool
	DecidedAt time.Time
}

func (s Snapshot) Restore() Departure {
	return Departure{leaving: s.Leaving, decidedAt: s.DecidedAt}
}

type Store interface {
	// The zero Departure for an account nobody asked to delete.
	Departure(ctx context.Context, account players.AccountID) (Departure, error)
	Change(ctx context.Context, account players.AccountID, change func(Departure) Departure) error
	Forget(ctx context.Context, account players.AccountID) error
}
