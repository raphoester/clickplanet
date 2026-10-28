package replays

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

var ErrInvalidRequest = errors.New("the replay asked for is invalid")

// Until zero is now.
type Request struct {
	Span  time.Duration
	Until time.Time
}

func (r Request) Window(now time.Time) (Window, error) {
	if r.Span <= 0 {
		return Window{}, fmt.Errorf("%w: it spans no time", ErrInvalidRequest)
	}

	until := r.Until
	if until.IsZero() {
		until = now
	}
	return Window{Since: until.Add(-r.Span), Until: until}, nil
}

type Window struct {
	Since time.Time
	Until time.Time
}

type Replay struct {
	// A planet.v1.GetReplayResponse in JSON, as the backend answered it.
	Document []byte
	Events   int
	Window   Window
}

type Source interface {
	Replay(ctx context.Context, window Window) (Replay, error)
}

type Store interface {
	Save(ctx context.Context, replay Replay) error
}

// The gamemaster's turn: the generator tells the moment scoring scored last (its debrief) when it ended in the replay,
// and calls countries up before the season's finale.
type Turn struct {
	// A gamemaster.v1.GetTurnResponse in JSON, as the backend answered it.
	Document []byte
	// The debrief's round as its season and its number, "1.5", "1.battle.2" or "1.finale"; empty when none.
	Debriefed string
	// The round to read the debrief's numbers by; empty when none.
	Round   RoundID
	EndedAt time.Time
	// When the finale battle to come starts; zero when none is scheduled.
	FinaleAt time.Time
}

type RoundID string

var ErrInvalidRoundID = errors.New("a round id is letters, digits, dashes and slashes")

func RoundIDOf(id string) (RoundID, error) {
	if id == "" {
		return "", fmt.Errorf("%w: it is empty", ErrInvalidRoundID)
	}
	for _, r := range id {
		if !strings.ContainsRune(roundIDRunes, r) {
			return "", fmt.Errorf("%w: %q", ErrInvalidRoundID, id)
		}
	}
	return RoundID(id), nil
}

const roundIDRunes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-/"

func (r RoundID) String() string { return string(r) }

type TurnSource interface {
	Turn(ctx context.Context) (Turn, error)
}

type TurnStore interface {
	SaveTurn(ctx context.Context, turn Turn) error
}

// The debrief's round as scoring scored it: its podium and the season's table after it.
type Round struct {
	// A scoring.v1.GetRoundResponse in JSON, as the backend answered it; {} when no round was debriefed.
	Document []byte
	ID       RoundID
}

var NoRound = Round{Document: []byte("{}")}

type RoundSource interface {
	Round(ctx context.Context, round RoundID) (Round, error)
}

type RoundStore interface {
	SaveRound(ctx context.Context, round Round) error
}
