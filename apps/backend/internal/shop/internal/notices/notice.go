package notices

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/payments"
	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/renewals"
)

var (
	ErrNotStripes  = errors.New("a notice Stripe did not sign, or signed more than 5 minutes ago")
	ErrInvalidNote = errors.New("not a notice of Stripe's")
)

// EventID is the Stripe event a notice came in (evt_): Stripe may send one again, and the shop acts on it once.
type EventID string

var eventShape = regexp.MustCompile(`^evt_[A-Za-z0-9_]+$`)

func ParseEventID(s string) (EventID, error) {
	if !eventShape.MatchString(s) {
		return "", fmt.Errorf("%w: the event %q", ErrInvalidNote, s)
	}
	return EventID(s), nil
}

func (id EventID) String() string {
	return string(id)
}

type Kind string

const (
	// A payment went through: a week's session, or an invoice of a month or a year.
	Paid Kind = "paid"
	// A payment refunded in full, or a dispute lost: its time goes back.
	TakenBack Kind = "taken_back"
	// A subscription began, changed or ended.
	Renewal Kind = "renewal"
	// What the shop does not act on: another kind of event, or what the shop never sold.
	Ignored Kind = "ignored"
)

// Notice is what one Stripe event tells the shop, in the shop's words.
type Notice struct {
	Event EventID
	Kind  Kind
	// Paid. An invoice's intent is read from Stripe: its event does not name it.
	Payment payments.Charged
	Invoice bool
	// TakenBack.
	Intent payments.IntentID
	At     time.Time
	// Renewal.
	Renewal renewals.Told
	// Ignored: why, for the log.
	Why string
}

// Reader checks that Stripe signed a notice, and reads it. ErrNotStripes for one it did not.
type Reader interface {
	Read(payload []byte, signature string) (Notice, error)
}

// Inbox keeps each event the shop acted on, in the transaction of what it did.
type Inbox interface {
	// Fresh is false for an event kept already: told again, it changes nothing.
	Keep(ctx context.Context, id EventID, at time.Time) (fresh bool, err error)
}
