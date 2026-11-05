package messages

import (
	"context"
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
	rooms     Rooms
}

func NewDrafts(ids IDProvider, clock cptime.Clock, limits Limits, countries Countries, rooms Rooms) Drafts {
	return Drafts{ids: ids, clock: clock, limits: limits, countries: countries, rooms: rooms}
}

func (d Drafts) Draft(
	ctx context.Context,
	account AccountID,
	room Room,
	country string,
	text string,
	gif Gif,
) (Message, error) {
	cleaned, shown, err := d.limits.Body(text, gif)
	if err != nil {
		return Message{}, fmt.Errorf("failed to check the message: %w", err)
	}
	if !d.countries.CheckCountry(country) {
		return Message{}, fmt.Errorf("%w: invalid country code %q", ErrInvalidMessage, country)
	}
	if err := d.rooms.EntryError(ctx, account, room); err != nil {
		return Message{}, err
	}

	id, err := d.ids.NewID()
	if err != nil {
		return Message{}, fmt.Errorf("failed to draw a message id: %w", err)
	}

	return Snapshot{
		ID: id, SentAt: d.clock.Now(), Account: account, Room: room, Country: country, Text: cleaned, Gif: shown,
	}.Restore()
}
