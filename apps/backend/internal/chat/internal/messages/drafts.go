package messages

import (
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Countries interface {
	CheckCountry(country string) bool
}

type Drafts struct {
	ids       IDProvider
	clock     cptime.Clock
	limits    Limits
	countries Countries
}

func NewDrafts(ids IDProvider, clock cptime.Clock, limits Limits, countries Countries) Drafts {
	return Drafts{ids: ids, clock: clock, limits: limits, countries: countries}
}

func (d Drafts) Draft(account AccountID, country string, text string) (Message, error) {
	cleaned, err := d.limits.Text(text)
	if err != nil {
		return Message{}, fmt.Errorf("failed to check the message: %w", err)
	}
	if !d.countries.CheckCountry(country) {
		return Message{}, fmt.Errorf("%w: invalid country code %q", ErrInvalidMessage, country)
	}

	id, err := d.ids.NewID()
	if err != nil {
		return Message{}, fmt.Errorf("failed to draw a message id: %w", err)
	}

	return NewMessage(id, d.clock.Now(), account, country, cleaned), nil
}
