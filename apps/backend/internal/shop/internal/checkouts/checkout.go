// Package checkouts keeps each Stripe page the shop opened, so a flag rejected meanwhile closes its page before it is
// paid.
package checkouts

import (
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/orders"
)

var ErrInvalidCheckout = errors.New("not a Checkout Session of Stripe's")

// SessionID is a Stripe Checkout Session (cs_): its page is open until paid, expired, or a day old.
type SessionID string

var sessionShape = regexp.MustCompile(`^cs_[A-Za-z0-9_]+$`)

func ParseSessionID(s string) (SessionID, error) {
	if !sessionShape.MatchString(s) {
		return "", fmt.Errorf("%w: %q", ErrInvalidCheckout, s)
	}
	return SessionID(s), nil
}

func (id SessionID) String() string {
	return string(id)
}

// Page is a Checkout Session as Stripe answers it: its id, and the page the buyer goes to.
type Page struct {
	Session SessionID
	URL     string
}

// Checkout is a page opened for a target.
type Checkout struct {
	session  SessionID
	target   orders.Target
	openedAt time.Time
}

func Opened(page Page, target orders.Target, at time.Time) Checkout {
	return Checkout{session: page.Session, target: target, openedAt: at}
}

// Stripe closes a Checkout Session's page after a day.
const pageLife = 24 * time.Hour

// Expirable is the session to expire while its page may still be open.
func (c Checkout) Expirable(now time.Time) (SessionID, bool) {
	return c.session, now.Sub(c.openedAt) < pageLife
}

func (c Checkout) TakeSnapshot() Snapshot {
	return Snapshot{Session: c.session, Target: c.target, OpenedAt: c.openedAt}
}

type Snapshot struct {
	Session  SessionID
	Target   orders.Target
	OpenedAt time.Time
}

func (s Snapshot) Restore() (Checkout, error) {
	if _, err := ParseSessionID(s.Session.String()); err != nil {
		return Checkout{}, err
	}
	return Checkout{session: s.Session, target: s.Target, openedAt: s.OpenedAt}, nil
}
