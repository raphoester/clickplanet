package play_scoring_usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Kept interface {
	KeepOpened(ctx context.Context, round moments.RoundID) error
	KeepRound(ctx context.Context, scored moments.ScoredRound) (bool, error)
	KeepBook(ctx context.Context, closed moments.ClosedBook) (bool, error)
	KeepMissedRound(ctx context.Context, missed moments.MissedRound) (bool, error)
	KeepMissedBook(ctx context.Context, missed moments.MissedBook) (bool, error)
}

// Scoring answers a command, or refuses it for good with moments.ErrRefused; any other error is no answer.
type Scoring interface {
	Ask(ctx context.Context, command moments.Command) (moments.Answer, error)
}

// Call is a command scoring answered this time: its answer, or its refusal for good.
type Call struct {
	Command moments.Command
	Answer  moments.Answer
	Refusal error
}

type Executor interface {
	Execute(ctx context.Context, plan moments.Plan) ([]Call, error)
}

func New(
	kept Kept, scoring Scoring, events moments.Events, transactioner cpatomic.Transactioner, clock cptime.Clock,
	again time.Duration,
) *UseCase {
	return &UseCase{
		kept: kept, scoring: scoring, events: events, transactioner: transactioner, clock: clock,
		calls: moments.NewCalls[moments.Key](again),
	}
}

// UseCase calls scoring with each command due, in order, until it answers, and keeps each close's answer right after
// the call. No call goes before the ones ahead of it are answered: a round opens after its book, a book closes after its
// rounds.
type UseCase struct {
	kept          Kept
	scoring       Scoring
	events        moments.Events
	transactioner cpatomic.Transactioner
	clock         cptime.Clock

	calls *moments.Calls[moments.Key]
}

var _ Executor = (*UseCase)(nil)

func (u *UseCase) Execute(ctx context.Context, plan moments.Plan) ([]Call, error) {
	now := u.clock.Now()
	var calls []Call
	for _, command := range plan.DueAt(now) {
		key := command.Key()
		if u.calls.Answered(key) {
			continue
		}
		if u.calls.Waits(key, now) {
			return calls, nil
		}
		answer, err := u.call(ctx, plan, command)
		if errors.Is(err, moments.ErrRefused) {
			u.calls.Answer(key)
			calls = append(calls, Call{Command: command, Refusal: err})
			continue
		}
		if err != nil {
			u.calls.Fail(key, now)
			return calls, err
		}
		u.calls.Answer(key)
		calls = append(calls, Call{Command: command, Answer: answer})
	}
	return calls, nil
}

func (u *UseCase) call(ctx context.Context, plan moments.Plan, command moments.Command) (moments.Answer, error) {
	if command.Action == moments.OpenRound {
		if err := u.kept.KeepOpened(ctx, command.Round); err != nil {
			return moments.Answer{}, err //nolint:wrapcheck // the store names the round.
		}
	}
	answer, err := u.scoring.Ask(ctx, command)
	if err != nil {
		return moments.Answer{}, err //nolint:wrapcheck // the adapter names the command.
	}
	return answer, u.keep(ctx, plan, command, answer)
}

func (u *UseCase) keep(ctx context.Context, plan moments.Plan, command moments.Command, answer moments.Answer) error {
	switch command.Action {
	case moments.OpenBook, moments.OpenRound:
		return nil
	case moments.CloseRound:
		if answer.Missed {
			_, err := u.kept.KeepMissedRound(ctx, moments.MissedRound{Round: command.Round, EndedAt: answer.EndedAt})
			return err //nolint:wrapcheck // the store names the round.
		}
		moment, err := plan.RoundEnded(command.Round, answer.EndedAt)
		if err != nil {
			return err //nolint:wrapcheck // the plan names the round.
		}
		return u.tell(ctx, moment, func(ctx context.Context) (bool, error) {
			return u.kept.KeepRound(ctx, moments.ScoredRound{Round: command.Round, Book: command.Book, EndedAt: answer.EndedAt})
		})
	case moments.CloseBook:
		if answer.Missed {
			_, err := u.kept.KeepMissedBook(ctx, moments.MissedBook{Book: command.Book, EndedAt: answer.EndedAt})
			return err //nolint:wrapcheck // the store names the book.
		}
		moment, err := plan.BookClosed(command.Book, answer.EndedAt)
		if err != nil {
			return err //nolint:wrapcheck // the plan names the book.
		}
		return u.tell(ctx, moment, func(ctx context.Context) (bool, error) {
			return u.kept.KeepBook(ctx, moments.ClosedBook{Book: command.Book, EndedAt: answer.EndedAt})
		})
	}
	return fmt.Errorf("scoring has no command to %s", command.Action)
}

func (u *UseCase) tell(ctx context.Context, moment moments.Moment, keep func(ctx context.Context) (bool, error)) error {
	return u.transactioner.InTx(ctx, func(ctx context.Context) error { //nolint:wrapcheck // the transaction's own error.
		kept, err := keep(ctx)
		if err != nil || !kept {
			return err //nolint:wrapcheck // the store names it.
		}
		return u.events.Record(ctx, moment) //nolint:wrapcheck // the adapter names the event.
	})
}
