package messages

import (
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Drafts struct {
	ids    IDProvider
	clock  cptime.Clock
	limits Limits
}

func NewDrafts(ids IDProvider, clock cptime.Clock, limits Limits) Drafts {
	return Drafts{ids: ids, clock: clock, limits: limits}
}

func (d Drafts) Draft(account AccountID, country string, text string) (Message, error) {
	cleaned, err := d.limits.Text(text)
	if err != nil {
		return Message{}, fmt.Errorf("failed to check the message: %w", err)
	}

	id, err := d.ids.NewID()
	if err != nil {
		return Message{}, fmt.Errorf("failed to draw a message id: %w", err)
	}

	return NewMessage(id, d.clock.Now(), account, country, cleaned), nil
}
