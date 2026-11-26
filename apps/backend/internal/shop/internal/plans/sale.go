package plans

import (
	"errors"
	"fmt"
	"regexp"
	"time"
)

var ErrInvalidSale = errors.New("not a sale")

// CouponID is a Stripe coupon, made by the owner in Stripe's Dashboard.
type CouponID string

var couponIDShape = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

func (id CouponID) String() string {
	return string(id)
}

// Sale takes a Stripe coupon off every plan's first payment, from one moment until another: a week is paid once, and a
// month or a year renews at its full price. The zero Sale is none: it never runs.
type Sale struct {
	coupon     CouponID
	percentOff uint32
	from       time.Time
	until      time.Time
}

// SaleSnapshot is a sale as the config gives it and the page is told of it.
type SaleSnapshot struct {
	// The coupon at Stripe, limited to the flags' products and redeemable until the sale's end.
	Coupon CouponID
	// What the coupon takes off, as the page shows it: 50 is half.
	PercentOff uint32
	From       time.Time
	// The sale's end, excluded.
	Until time.Time
}

func (s SaleSnapshot) Restore() (Sale, error) {
	if !couponIDShape.MatchString(s.Coupon.String()) {
		return Sale{}, fmt.Errorf("%w: the coupon %q is no Stripe coupon id", ErrInvalidSale, s.Coupon)
	}
	if s.PercentOff == 0 || s.PercentOff >= 100 {
		return Sale{}, fmt.Errorf("%w: %d%% off is not between 1 and 99", ErrInvalidSale, s.PercentOff)
	}
	if s.From.IsZero() || !s.From.Before(s.Until) {
		return Sale{}, fmt.Errorf("%w: from %s until %s", ErrInvalidSale, s.From, s.Until)
	}
	return Sale{coupon: s.Coupon, percentOff: s.PercentOff, from: s.From, until: s.Until}, nil
}

func (s Sale) TakeSnapshot() SaleSnapshot {
	return SaleSnapshot{Coupon: s.coupon, PercentOff: s.percentOff, From: s.from, Until: s.until}
}

// Runs is true from the sale's start until its end, the end excluded.
func (s Sale) Runs(at time.Time) bool {
	return !at.Before(s.from) && at.Before(s.until)
}

// CouponsAt are the coupons Stripe takes off a checkout made at that moment: the sale's while it runs, else none.
func (s Sale) CouponsAt(at time.Time) []CouponID {
	if !s.Runs(at) {
		return []CouponID{}
	}
	return []CouponID{s.coupon}
}
