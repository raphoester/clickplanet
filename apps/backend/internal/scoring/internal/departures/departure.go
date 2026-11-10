package departures

import (
	"context"
	"errors"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
)

type AccountID = cpsession.AccountID

var ErrNoTime = errors.New("the decision says no time")

// An account its player asked to delete: no board ranks it until player forgets it, or it returns.
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

func (d Decision) TimeError() error {
	if d.at.IsZero() {
		return ErrNoTime
	}
	return nil
}

// Only a later decision changes a departure: player tells a leave and a return on two streams, which may come in
// either order, and either may come twice.
func (d Departure) Decided(decision Decision) Departure {
	if !decision.at.After(d.decidedAt) {
		return d
	}
	return Departure{leaving: decision.leaving, decidedAt: decision.at}
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
	Change(ctx context.Context, account AccountID, change func(Departure) Departure) error
	Forget(ctx context.Context, account AccountID) error
}
