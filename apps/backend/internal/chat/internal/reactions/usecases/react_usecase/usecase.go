package react_usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/feed"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/reactions"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Messages interface {
	Message(ctx context.Context, account messages.AccountID, id messages.MessageID) (messages.Message, error)
}

type Board interface {
	Save(ctx context.Context, change reactions.Change) error
	Reactions(ctx context.Context, ids []messages.MessageID) (map[messages.MessageID]reactions.Reactions, error)
}

type Publisher interface {
	Publish(update feed.Update)
}

type In struct {
	Account   messages.AccountID
	MessageID messages.MessageID
	Reaction  reactions.Reaction
	On        bool
}

const writeTimeout = 5 * time.Second

func New(shown Messages, board Board, publisher Publisher, clock cptime.Clock) *UseCase {
	return &UseCase{shown: shown, board: board, publisher: publisher, clock: clock}
}

type UseCase struct {
	shown     Messages
	board     Board
	publisher Publisher
	clock     cptime.Clock
}

type Out struct {
	Counts  []reactions.Count
	Version uint64
}

func (u *UseCase) Execute(ctx context.Context, in In) (Out, error) {
	reactor := reactions.ReactorOf(in.Account)
	if reactor == reactions.NoReactor {
		return Out{}, messages.ErrNoAccount
	}

	ctx, cancel := context.WithTimeout(ctx, writeTimeout)
	defer cancel()

	shown, err := u.shown.Message(ctx, in.Account, in.MessageID)
	if errors.Is(err, messages.ErrNotShown) {
		return Out{}, fmt.Errorf("%w: %q", reactions.ErrUnknownMessage, in.MessageID)
	}
	if err != nil {
		return Out{}, fmt.Errorf("failed to read the message: %w", err)
	}

	current, err := u.of(ctx, in.MessageID)
	if err != nil {
		return Out{}, err
	}
	if current.Given(in.Reaction, reactor) == in.On {
		return Out{Counts: current.Tally(reactor), Version: current.Version()}, nil
	}

	change := reactions.Off(in.MessageID, in.Reaction, reactor, u.clock.Now())
	if in.On {
		change = reactions.On(in.MessageID, in.Reaction, reactor, u.clock.Now())
	}
	if err := u.board.Save(ctx, change); err != nil {
		return Out{}, fmt.Errorf("failed to save the reaction: %w", err)
	}

	next, err := u.of(ctx, in.MessageID)
	if err != nil {
		return Out{}, err
	}
	u.publisher.Publish(feed.ReactionsChanged(shown, next.TallyOf(in.MessageID)))

	return Out{Counts: next.Tally(reactor), Version: next.Version()}, nil
}

func (u *UseCase) of(ctx context.Context, id messages.MessageID) (reactions.Reactions, error) {
	given, err := u.board.Reactions(ctx, []messages.MessageID{id})
	if err != nil {
		return reactions.Reactions{}, fmt.Errorf("failed to read the reactions: %w", err)
	}
	return given[id], nil
}
