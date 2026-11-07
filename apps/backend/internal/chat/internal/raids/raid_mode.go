package raids

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/moderation"
)

// Never longer: a guest who never raided is refused too, and longer is the operator's call, run after run.
const MaxDuration = 24 * time.Hour

var (
	ErrInvalidDuration = errors.New("raid mode runs whole seconds, 24 hours at most")
	ErrInvalidRaidMode = errors.New("the raid mode is invalid")
	ErrNoRaidMode      = errors.New("no raid mode was ever set")
	ErrGuest           = errors.New("the chat takes no guest's post while raid mode runs")
)

// While raid mode runs, the chat takes no guest's post. A guest account is free and a private tab is a new one, so a
// mute holds a raid of guests no longer than it takes to open a tab; a signed-in account costs an account of Google,
// Discord or an address. A signed-in player posts as before, and everyone still reads and reacts. One at a time, and
// it ends by itself.
type RaidMode struct {
	startedAt time.Time
	endsAt    time.Time
	reason    moderation.Reason
}

// 0 is no raid mode: an operator who asks for none ends the one that runs.
func DurationOf(asked time.Duration) (time.Duration, error) {
	if asked < 0 || asked > MaxDuration || asked != asked.Truncate(time.Second) {
		return 0, fmt.Errorf("%w: %s", ErrInvalidDuration, asked)
	}
	return asked, nil
}

func Start(at time.Time, duration time.Duration, reason moderation.Reason) (RaidMode, error) {
	if duration <= 0 {
		return RaidMode{}, fmt.Errorf("%w: %s", ErrInvalidDuration, duration)
	}
	return Snapshot{StartedAt: at, EndsAt: at.Add(duration), Reason: reason}.Restore()
}

// Ended now, if it would end later. The row stays: it says when the last one ran.
func (m RaidMode) Ended(at time.Time) RaidMode {
	if at.Before(m.endsAt) {
		m.endsAt = at
	}
	return m
}

func (m RaidMode) RunsAt(at time.Time) bool {
	return !at.Before(m.startedAt) && at.Before(m.endsAt)
}

// The whole rule on a post: who posts, and nothing it says.
func (m RaidMode) Admits(author messages.Author) error {
	if author.Guest() {
		return ErrGuest
	}
	return nil
}

func (m RaidMode) TakeSnapshot() Snapshot {
	return Snapshot{StartedAt: m.startedAt, EndsAt: m.endsAt, Reason: m.reason}
}

// The state of a RaidMode, for a store. It holds no rule: Restore does.
type Snapshot struct {
	StartedAt time.Time
	EndsAt    time.Time
	Reason    moderation.Reason
}

func (s Snapshot) Restore() (RaidMode, error) {
	switch {
	case s.StartedAt.IsZero():
		return RaidMode{}, fmt.Errorf("%w: no start", ErrInvalidRaidMode)
	case s.EndsAt.Before(s.StartedAt):
		return RaidMode{}, fmt.Errorf("%w: it ends before it starts", ErrInvalidRaidMode)
	case s.EndsAt.Sub(s.StartedAt) > MaxDuration:
		return RaidMode{}, fmt.Errorf("%w: it runs %s", ErrInvalidDuration, s.EndsAt.Sub(s.StartedAt))
	}
	return RaidMode{startedAt: s.StartedAt, endsAt: s.EndsAt, reason: s.Reason}, nil
}

// The one raid mode, running or ended: a raid mode saved replaces the one before.
type Storage interface {
	// ErrNoRaidMode when none was ever saved.
	Current(ctx context.Context) (RaidMode, error)
	Save(ctx context.Context, mode RaidMode) error
}
