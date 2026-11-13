package click_tried_subscriber

import (
	"context"
	"time"

	"github.com/google/uuid"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpredis"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
)

// Presence is the offers' feed of who plays.
type Presence interface {
	Clicked(click bonuses.Click)
}

func New(presence Presence) Subscriber {
	return Subscriber{presence: presence}
}

// Subscriber hears every click planet tried and keeps the ones it took, struck a shield with or found already wearing
// its flag: the clicks the throttle let through and the chain inside it answered. A refused, throttled, invalid or
// failed try, or one for another flag or out of reach, is no sign of play.
type Subscriber struct {
	presence Presence
}

var _ cpredis.Handler[*planetv1.ClickTried] = Subscriber{}

var played = map[planetv1.ClickOutcome]bool{
	planetv1.ClickOutcome_CLICK_OUTCOME_TAKEN:     true,
	planetv1.ClickOutcome_CLICK_OUTCOME_STRUCK:    true,
	planetv1.ClickOutcome_CLICK_OUTCOME_UNCHANGED: true,
}

func (s Subscriber) Handle(_ context.Context, event *planetv1.ClickTried) error {
	if !played[event.GetOutcome()] {
		return nil
	}

	s.presence.Clicked(bonuses.Click{
		Caller: bonuses.Caller{
			Scope: event.GetScope(), Account: bonuses.Account(event.GetAccountId()), Linked: event.GetLinked(),
		},
		At:   event.GetTriedAt().AsTime(),
		Born: bornOf(event.GetAccountId()),
	})
	return nil
}

// bornOf reads when an account was made off its id; zero for none, or for an id that does not say.
func bornOf(account string) time.Time {
	id, err := uuid.Parse(account)
	if err != nil {
		return time.Time{}
	}
	born, _ := cpsession.AccountID(id).CreatedAt()
	return born
}
