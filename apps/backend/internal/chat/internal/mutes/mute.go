package mutes

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpipscope"
)

type MuteID uuid.UUID

type Scope string

const NoScope Scope = ""

func ScopeOf(ip string) Scope {
	scope, ok := cpipscope.Parse(ip)
	if !ok {
		return NoScope
	}
	return Scope(scope)
}

type Caller struct {
	account messages.AccountID
	scope   Scope
}

func NewCaller(account messages.AccountID, ip string) Caller {
	return Caller{account: account, scope: ScopeOf(ip)}
}

func CallerOf(account messages.AccountID, scope Scope) Caller {
	return Caller{account: account, scope: scope}
}

func (c Caller) Account() messages.AccountID { return c.account }

func (c Caller) Scope() Scope { return c.scope }

const DefaultDuration = time.Hour

var (
	ErrNoAccount       = errors.New("a mute names an account")
	ErrInvalidDuration = errors.New("a mute lasts a whole number of seconds")
	ErrNotMuted        = errors.New("no mute applies to this caller")
	ErrMuted           = errors.New("muted in the chat")
)

func DurationOf(asked time.Duration) (time.Duration, error) {
	switch {
	case asked == 0:
		return DefaultDuration, nil
	case asked < 0 || asked != asked.Truncate(time.Second):
		return 0, fmt.Errorf("%w: %s", ErrInvalidDuration, asked)
	default:
		return asked, nil
	}
}

type Mute struct {
	id     MuteID
	caller Caller
	at     time.Time
	until  time.Time
}

func NewMute(id MuteID, caller Caller, at time.Time, duration time.Duration) Mute {
	return Mute{id: id, caller: caller, at: at, until: at.Add(duration)}
}

func MuteOf(id MuteID, caller Caller, at time.Time, until time.Time) Mute {
	return Mute{id: id, caller: caller, at: at, until: until}
}

func (m Mute) ID() MuteID { return m.id }

func (m Mute) Caller() Caller { return m.caller }

func (m Mute) At() time.Time { return m.at }

func (m Mute) Until() time.Time { return m.until }

func (m Mute) Duration() time.Duration { return m.until.Sub(m.at) }

func (m Mute) ApplicableTo(caller Caller, at time.Time) bool {
	if !at.Before(m.until) {
		return false
	}
	if m.caller.account == caller.account {
		return true
	}
	return m.caller.scope != NoScope && m.caller.scope == caller.scope
}

func (m Mute) Refusal() Refusal { return Refusal{until: m.until} }

type Refusal struct {
	until time.Time
}

func (r Refusal) Error() string {
	return fmt.Sprintf("%s until %s", ErrMuted, r.until.UTC().Format(time.RFC3339))
}

func (r Refusal) Unwrap() error { return ErrMuted }

func (r Refusal) Until() time.Time { return r.until }

type Storage interface {
	Save(ctx context.Context, mute Mute) error
	Mute(ctx context.Context, caller Caller, at time.Time) (Mute, error)
	DeleteBefore(ctx context.Context, cutoff time.Time) (int64, error)
}
