package plans

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

var (
	ErrInvalidPlan  = errors.New("not a plan on sale")
	ErrInvalidPrice = errors.New("not a price of Stripe's")
	ErrOff          = errors.New("the shop is off")
)

// Plan is what a buyer pays for: a week paid once, or a month or a year that renews.
type Plan string

const (
	Week  Plan = "week"
	Month Plan = "month"
	Year  Plan = "year"
)

func ParsePlan(s string) (Plan, error) {
	plan := Plan(s)
	if err := plan.check(); err != nil {
		return "", err
	}
	return plan, nil
}

func (p Plan) check() error {
	switch p {
	case Week, Month, Year:
		return nil
	}
	return fmt.Errorf("%w: %q", ErrInvalidPlan, p)
}

// Renews is true for a plan paid again at each period's end, until it is stopped.
func (p Plan) Renews() bool {
	return p == Month || p == Year
}

// PriceID is a Stripe price, made by the owner in Stripe's Dashboard: price_ and letters or digits.
type PriceID string

var priceIDShape = regexp.MustCompile(`^price_[A-Za-z0-9]+$`)

func ParsePriceID(s string) (PriceID, error) {
	if !priceIDShape.MatchString(s) {
		return "", fmt.Errorf("%w: %q", ErrInvalidPrice, s)
	}
	return PriceID(s), nil
}

func (id PriceID) String() string {
	return string(id)
}

// Price is a plan's price as the page shows it: what Stripe's price charges, taxes included.
type Price struct {
	Plan Plan
	ID   PriceID
	// In the currency's smallest unit.
	Amount uint32
	// ISO 4217, lower case.
	Currency string
}

var currencyShape = regexp.MustCompile(`^[a-z]{3}$`)

func (p Price) check() error {
	if err := p.Plan.check(); err != nil {
		return err
	}
	if _, err := ParsePriceID(p.ID.String()); err != nil {
		return err
	}
	if p.Amount == 0 || !currencyShape.MatchString(p.Currency) {
		return fmt.Errorf("%w: %s at %d %q", ErrInvalidPrice, p.Plan, p.Amount, p.Currency)
	}
	return nil
}

// Catalog is the plans on sale, in the order the page shows them. Empty, the shop is off.
type Catalog struct {
	prices []Price
}

// NewCatalog is the three plans, each with its price, or none: a shop sells all three or nothing.
func NewCatalog(prices ...Price) (Catalog, error) {
	if len(prices) == 0 {
		return Catalog{}, nil
	}
	var errs []error
	for _, plan := range []Plan{Week, Month, Year} {
		found := 0
		for _, price := range prices {
			if price.Plan == plan {
				found++
			}
		}
		if found != 1 {
			errs = append(errs, fmt.Errorf("%w: the %s has %d prices", ErrInvalidPrice, plan, found))
		}
	}
	for _, price := range prices {
		if err := price.check(); err != nil {
			errs = append(errs, err)
		}
	}
	if err := errors.Join(errs...); err != nil {
		return Catalog{}, err
	}
	sorted := make([]Price, 0, len(prices))
	for _, plan := range []Plan{Week, Month, Year} {
		for _, price := range prices {
			if price.Plan == plan {
				sorted = append(sorted, Price{
					Plan: price.Plan, ID: price.ID, Amount: price.Amount, Currency: strings.ToLower(price.Currency),
				})
			}
		}
	}
	return Catalog{prices: sorted}, nil
}

// Prices is every plan on sale: none while the shop is off.
func (c Catalog) Prices() []Price {
	return append([]Price(nil), c.prices...)
}

// Price is the plan's, ErrOff while the shop is off.
func (c Catalog) Price(plan Plan) (Price, error) {
	for _, price := range c.prices {
		if price.Plan == plan {
			return price, nil
		}
	}
	if len(c.prices) == 0 {
		return Price{}, ErrOff
	}
	return Price{}, fmt.Errorf("%w: %q", ErrInvalidPlan, plan)
}
