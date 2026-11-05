package hear_round_usecase

import (
	"context"
	"errors"

	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
)

type Planner interface {
	Plan(ctx context.Context) (moments.Plan, error)
}

type Rounds interface {
	KeepRound(ctx context.Context, scored moments.ScoredRound) (bool, error)
}

// Heard is what came of a round scoring told.
type Heard uint8

const (
	// Told: the round was new, and its moment is recorded.
	Told Heard = iota + 1
	// Repeated: the round was kept already.
	Repeated
	// Unknown: no day or battle has this round.
	Unknown
)

type UseCase interface {
	Execute(ctx context.Context, scored moments.ScoredRound) (Heard, error)
}

func New(planner Planner, rounds Rounds, events moments.Events, transactioner cpatomic.Transactioner) *Hearing {
	return &Hearing{planner: planner, rounds: rounds, events: events, transactioner: transactioner}
}

// Hearing keeps each round scoring scored, and tells its moment once, in the game's words.
type Hearing struct {
	planner       Planner
	rounds        Rounds
	events        moments.Events
	transactioner cpatomic.Transactioner
}

func (h *Hearing) Execute(ctx context.Context, scored moments.ScoredRound) (Heard, error) {
	plan, err := h.planner.Plan(ctx)
	if err != nil {
		return 0, err //nolint:wrapcheck // the planner names what it read.
	}
	moment, err := plan.RoundEnded(scored.Round, scored.EndedAt)
	if errors.Is(err, moments.ErrUnknownRound) {
		return Unknown, nil
	}
	if err != nil {
		return 0, err //nolint:wrapcheck // the plan names the round.
	}

	heard := Repeated
	err = h.transactioner.InTx(ctx, func(ctx context.Context) error {
		kept, err := h.rounds.KeepRound(ctx, scored)
		if err != nil || !kept {
			return err //nolint:wrapcheck // the store names the round.
		}
		heard = Told
		return h.events.Record(ctx, moment) //nolint:wrapcheck // the adapter names the event.
	})
	if err != nil {
		return 0, err //nolint:wrapcheck // the transaction's own error.
	}
	return heard, nil
}
