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

func (id MuteID) String() string { return uuid.UUID(id).String() }

var ErrInvalidMute = errors.New("not a mute id")

func MuteIDOf(value string) (MuteID, error) {
	id, err := uuid.Parse(value)
	if err != nil {
		return MuteID{}, fmt.Errorf("%w: %q", ErrInvalidMute, value)
	}
	return MuteID(id), nil
}

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

// What a mute takes away. The zero value is the whole chat, as on the wire.
type Kind uint8

const (
	// Every post and every reaction.
	Chat Kind = iota
	// A post with a GIF: the player still writes and reacts.
	Gifs
)

// As postgres keeps it: a value of the type mute_kind.
func (k Kind) String() string {
	if k == Gifs {
		return "gifs"
	}
	return "chat"
}

const DefaultDuration = time.Hour

var (
	ErrNoAccount       = errors.New("a mute names an account")
	ErrInvalidKind     = errors.New("not a kind of mute")
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
	kind   Kind
	at     time.Time
	until  time.Time
}

func NewMute(id MuteID, caller Caller, kind Kind, at time.Time, duration time.Duration) Mute {
	return Mute{id: id, caller: caller, kind: kind, at: at, until: at.Add(duration)}
}

func MuteOf(id MuteID, caller Caller, kind Kind, at time.Time, until time.Time) Mute {
	return Mute{id: id, caller: caller, kind: kind, at: at, until: until}
}

func (m Mute) ID() MuteID { return m.id }

func (m Mute) Caller() Caller { return m.caller }

func (m Mute) Kind() Kind { return m.kind }

func (m Mute) At() time.Time { return m.at }

func (m Mute) Until() time.Time { return m.until }

func (m Mute) Duration() time.Duration { return m.until.Sub(m.at) }

func (m Mute) Ended(at time.Time) bool { return !at.Before(m.until) }

func (m Mute) Lifted(at time.Time) Mute {
	m.until = at
	return m
}

func (m Mute) ApplicableTo(caller Caller, kind Kind, at time.Time) bool {
	if m.kind != kind || m.Ended(at) {
		return false
	}
	if m.caller.account == caller.account {
		return true
	}
	return m.caller.scope != NoScope && m.caller.scope == caller.scope
}

func (m Mute) Refusal() Refusal { return Refusal{kind: m.kind, until: m.until} }

type Refusal struct {
	kind  Kind
	until time.Time
}

func (r Refusal) Error() string {
	return fmt.Sprintf("%s (%s) until %s", ErrMuted, r.kind, r.until.UTC().Format(time.RFC3339))
}

func (r Refusal) Kind() Kind { return r.kind }

func (r Refusal) Unwrap() error { return ErrMuted }

func (r Refusal) Until() time.Time { return r.until }

type Storage interface {
	Save(ctx context.Context, mute Mute) error
	Mute(ctx context.Context, caller Caller, kind Kind, at time.Time) (Mute, error)
	MutesOf(ctx context.Context, account messages.AccountID, kind Kind) ([]Mute, error)
	DeleteBefore(ctx context.Context, cutoff time.Time) (int64, error)
}
