package scoring_command_handler

import (
	"context"
	"errors"
	"fmt"

	"google.golang.org/protobuf/proto"

	scoringv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/scoring/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/books"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/books/usecases/close_book_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/books/usecases/close_round_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/books/usecases/open_book_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/books/usecases/open_round_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/subscribers"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpredis"
)

type Executor[C any] interface {
	Execute(ctx context.Context, command C) (bool, error)
}

type UseCases struct {
	OpenBook   Executor[open_book_usecase.Command]
	OpenRound  Executor[open_round_usecase.Command]
	CloseRound Executor[close_round_usecase.Command]
	CloseBook  Executor[close_book_usecase.Command]
}

func NewDispatcher(useCases UseCases) Dispatcher {
	return Dispatcher{useCases: useCases}
}

// Dispatcher hands each command to its use case, and answers every error, a refusal too.
type Dispatcher struct {
	useCases UseCases
}

var _ cpredis.Handler[*scoringv1.ScoringCommand] = Dispatcher{}

var (
	ErrNoCommand = errors.New("the scoring command names no command")
	errNoTime    = errors.New("the command has no time")
)

func New[C proto.Message](inner cpredis.Handler[C]) Handler[C] {
	return Handler[C]{inner: inner}
}

// Handler drops a command refused: delivered again, it would hold back every command after it on its stream.
type Handler[C proto.Message] struct {
	inner cpredis.Handler[C]
}

var _ cpredis.Handler[*scoringv1.ScoringCommand] = Handler[*scoringv1.ScoringCommand]{}

func (h Handler[C]) Handle(ctx context.Context, command C) error {
	if err := h.inner.Handle(ctx, command); err != nil && !errors.Is(err, books.ErrRefused) {
		return err //nolint:wrapcheck // the dispatcher named it.
	}
	return nil
}

func (h Dispatcher) Handle(ctx context.Context, command *scoringv1.ScoringCommand) error {
	ctx, cancel := context.WithTimeout(ctx, subscribers.Timeout)
	defer cancel()

	return h.dispatch(ctx, command)
}

func (h Dispatcher) dispatch(ctx context.Context, command *scoringv1.ScoringCommand) error {
	switch kind := command.GetCommand().(type) {
	case *scoringv1.ScoringCommand_OpenBook:
		return h.openBook(ctx, kind.OpenBook)
	case *scoringv1.ScoringCommand_OpenRound:
		return h.openRound(ctx, kind.OpenRound)
	case *scoringv1.ScoringCommand_CloseRound:
		return h.closeRound(ctx, kind.CloseRound)
	case *scoringv1.ScoringCommand_CloseBook:
		return h.closeBook(ctx, kind.CloseBook)
	case nil:
		return refused(ErrNoCommand)
	}
	return refused(fmt.Errorf("%w: %T", ErrNoCommand, command.GetCommand()))
}

type timed interface {
	CheckValid() error
}

func checkTime(at timed) error {
	if err := at.CheckValid(); err != nil {
		return fmt.Errorf("%w: %w: %w", books.ErrRefused, errNoTime, err)
	}
	return nil
}

func refused(err error) error {
	return fmt.Errorf("%w: %w", books.ErrRefused, err)
}

func (h Dispatcher) openBook(ctx context.Context, command *scoringv1.OpenBook) error {
	book, err := books.ParseBookID(command.GetBookId())
	if err != nil {
		return refused(err)
	}
	if err := checkTime(command.GetStartsAt()); err != nil {
		return err
	}
	_, err = h.useCases.OpenBook.Execute(ctx, open_book_usecase.Command{Book: book, StartsAt: command.GetStartsAt().AsTime()})
	return err //nolint:wrapcheck // the use case named it.
}

func (h Dispatcher) openRound(ctx context.Context, command *scoringv1.OpenRound) error {
	round, err := books.ParseRoundID(command.GetRoundId())
	if err != nil {
		return refused(err)
	}
	book, err := books.ParseBookID(command.GetBookId())
	if err != nil {
		return refused(err)
	}
	if err := checkTime(command.GetStartsAt()); err != nil {
		return err
	}
	_, err = h.useCases.OpenRound.Execute(ctx, open_round_usecase.Command{
		Round: round, Book: book, Times: command.GetTimes(), StartsAt: command.GetStartsAt().AsTime(),
	})
	return err //nolint:wrapcheck // the use case named it.
}

func (h Dispatcher) closeRound(ctx context.Context, command *scoringv1.CloseRound) error {
	round, err := books.ParseRoundID(command.GetRoundId())
	if err != nil {
		return refused(err)
	}
	if err := checkTime(command.GetEndsAt()); err != nil {
		return err
	}
	_, err = h.useCases.CloseRound.Execute(ctx, close_round_usecase.Command{Round: round, EndsAt: command.GetEndsAt().AsTime()})
	return err //nolint:wrapcheck // the use case named it.
}

func (h Dispatcher) closeBook(ctx context.Context, command *scoringv1.CloseBook) error {
	book, err := books.ParseBookID(command.GetBookId())
	if err != nil {
		return refused(err)
	}
	if err := checkTime(command.GetEndsAt()); err != nil {
		return err
	}
	var tiebreak books.RoundID
	if command.GetTiebreakRoundId() != "" {
		if tiebreak, err = books.ParseRoundID(command.GetTiebreakRoundId()); err != nil {
			return refused(err)
		}
	}
	_, err = h.useCases.CloseBook.Execute(ctx, close_book_usecase.Command{
		Book: book, EndsAt: command.GetEndsAt().AsTime(), Tiebreak: tiebreak,
	})
	return err //nolint:wrapcheck // the use case named it.
}
