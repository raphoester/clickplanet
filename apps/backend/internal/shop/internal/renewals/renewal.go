package renewals

import (
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/orders"
	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/plans"
)

var (
	ErrUnknownRenewal = errors.New("no renewal was kept under this id")
	ErrInvalidRenewal = errors.New("not a subscription of Stripe's")
	ErrNotYours       = errors.New("only its buyer stops a renewal")
	ErrEnded          = errors.New("the renewal ended already")
	ErrRenewing       = errors.New("a renewal of it runs already")
	ErrInvalidState   = errors.New("a renewal kept in no state it can be in")
)

// SubscriptionID is the Stripe subscription that renews a month or a year (sub_).
type SubscriptionID string

var subscriptionShape = regexp.MustCompile(`^sub_[A-Za-z0-9_]+$`)

func ParseSubscriptionID(s string) (SubscriptionID, error) {
	if !subscriptionShape.MatchString(s) {
		return "", fmt.Errorf("%w: %q", ErrInvalidRenewal, s)
	}
	return SubscriptionID(s), nil
}

func (id SubscriptionID) String() string {
	return string(id)
}

type State string

const (
	Renewing State = "renewing"
	// Stopped by its buyer: no payment after the period paid.
	Stopping State = "stopping"
	// Over: stopped and past, cancelled, or never paid.
	Ended State = "ended"
)

// Renewal is a plan that renews, as Stripe tells of its subscription: the latest word holds.
type Renewal struct {
	id           SubscriptionID
	order        orders.Snapshot
	state        State
	periodEndsAt time.Time
	// When Stripe told it, or the buyer stopped it.
	decidedAt time.Time
}

// Told is a subscription as Stripe tells it.
type Told struct {
	ID           SubscriptionID
	Order        orders.Snapshot
	State        State
	PeriodEndsAt time.Time
	At           time.Time
}

// Heard is the renewal as Stripe told it last: what it told before the word kept changes nothing.
func Heard(kept Renewal, found bool, told Told) (Renewal, bool, error) {
	if found && told.At.Before(kept.decidedAt) {
		return kept, false, nil
	}
	renewal, err := Snapshot{
		ID: told.ID, Order: told.Order, State: told.State, PeriodEndsAt: told.PeriodEndsAt, DecidedAt: told.At,
	}.Restore()
	if err != nil {
		return kept, false, err
	}
	return renewal, renewal.TakeSnapshot() != kept.TakeSnapshot(), nil
}

// StoppedBy stops the renewal at the end of the period paid, for its buyer alone. One stopping stays so.
func (r Renewal) StoppedBy(buyer orders.AccountID, at time.Time) (Renewal, bool, error) {
	if r.order.Buyer != buyer {
		return r, false, fmt.Errorf("%w: %s", ErrNotYours, r.id)
	}
	switch r.state {
	case Renewing:
		r.state, r.decidedAt = Stopping, at
		return r, true, nil
	case Stopping:
		return r, false, nil
	case Ended:
	}
	return r, false, fmt.Errorf("%w: %s", ErrEnded, r.id)
}

// NoneRenewing is nil when no renewal runs for the target: a second one would charge its buyer twice. One stopping
// charges no more.
func NoneRenewing(kept []Renewal, target orders.Target) error {
	for _, renewal := range kept {
		if renewal.state == Renewing && renewal.order.Target == target {
			return fmt.Errorf("%w: %s", ErrRenewing, renewal.id)
		}
	}
	return nil
}

func (r Renewal) TakeSnapshot() Snapshot {
	return Snapshot{ID: r.id, Order: r.order, State: r.state, PeriodEndsAt: r.periodEndsAt, DecidedAt: r.decidedAt}
}

type Snapshot struct {
	ID           SubscriptionID
	Order        orders.Snapshot
	State        State
	PeriodEndsAt time.Time
	DecidedAt    time.Time
}

func (s Snapshot) Restore() (Renewal, error) {
	if _, err := ParseSubscriptionID(s.ID.String()); err != nil {
		return Renewal{}, err
	}
	if !s.Order.Plan.Renews() {
		return Renewal{}, fmt.Errorf("%w: %s renews a %q", ErrInvalidState, s.ID, s.Order.Plan)
	}
	if _, err := plans.ParsePlan(string(s.Order.Plan)); err != nil {
		return Renewal{}, fmt.Errorf("the renewal %s: %w", s.ID, err)
	}
	if _, err := orders.ParseTarget(s.Order.Target.Submission(), s.Order.Target.Flag()); err != nil {
		return Renewal{}, fmt.Errorf("the renewal %s: %w", s.ID, err)
	}
	switch s.State {
	case Renewing, Stopping, Ended:
	default:
		return Renewal{}, fmt.Errorf("%w: %s is %q", ErrInvalidState, s.ID, s.State)
	}
	return Renewal{id: s.ID, order: s.Order, state: s.State, periodEndsAt: s.PeriodEndsAt, decidedAt: s.DecidedAt}, nil
}
