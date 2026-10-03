package subscriptions

import (
	"errors"
	"time"
)

var (
	ErrNoSubscription      = errors.New("this account never asked for season emails")
	ErrNotLinked           = errors.New("only a signed-in account may ask for season emails")
	ErrAlreadySubscribed   = errors.New("this account already asked for season emails at another address")
	ErrAudienceUnreachable = errors.New("the mailing list could not be reached")
)

type State string

const (
	StateNone      State = "none"
	StateWaiting   State = "waiting"
	StateActive    State = "active"
	StateWithdrawn State = "withdrawn"
)

// The words on the button the player pressed; new words need a new version.
type Consent string

const SeasonEmails Consent = "season-emails-1"

type Subscription struct {
	Account     AccountID
	Address     Address
	State       State
	Consent     Consent
	AskedAt     time.Time
	WithdrawnAt time.Time
}

func NewSubscription(account Account, address Address, now time.Time) (*Subscription, error) {
	if !account.Linked {
		return nil, ErrNotLinked
	}

	state := StateWaiting
	if account.Verified(address) {
		state = StateActive
	}
	return &Subscription{Account: account.ID, Address: address, State: state, Consent: SeasonEmails, AskedAt: now}, nil
}

func (s *Subscription) Waiting() bool {
	return s.State == StateWaiting
}

func (s *Subscription) Live() bool {
	return s.State == StateWaiting || s.State == StateActive
}

func (s *Subscription) ChangeError(address Address) error {
	if !s.Live() || s.Address == address {
		return nil
	}
	return ErrAlreadySubscribed
}

func (s *Subscription) Confirmed() *Subscription {
	confirmed := *s
	confirmed.State = StateActive
	return &confirmed
}

func (s *Subscription) Withdrawn(now time.Time) *Subscription {
	withdrawn := *s
	withdrawn.State = StateWithdrawn
	withdrawn.WithdrawnAt = now
	return &withdrawn
}

type Status struct {
	State   State
	Address Address
}

func StatusOf(account Account) Status {
	return Status{State: StateNone, Address: account.Suggestion()}
}

func (s *Subscription) Status(account Account) Status {
	if !s.Live() {
		return StatusOf(account)
	}
	return Status{State: s.State, Address: s.Address}
}
