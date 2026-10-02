package react_usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/feed"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/reactions"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Messages interface {
	Shown(ctx context.Context, id messages.MessageID, since time.Time, limit int) (bool, error)
}

type Board interface {
	Save(ctx context.Context, change reactions.Change) error
	Reactions(ctx context.Context, ids []messages.MessageID) (map[messages.MessageID]reactions.Reactions, error)
}

type Publisher interface {
	Publish(update feed.Update)
}

type Authors interface {
	Authors(ctx context.Context, accounts []messages.AccountID) (map[messages.AccountID]messages.Author, error)
}

type In struct {
	Account   messages.AccountID
	MessageID messages.MessageID
	Reaction  reactions.Reaction
	On        bool
}

const writeTimeout = 5 * time.Second

func New(
	shown Messages,
	board Board,
	publisher Publisher,
	authors Authors,
	clock cptime.Clock,
	window messages.Window,
) *UseCase {
	return &UseCase{shown: shown, board: board, publisher: publisher, authors: authors, clock: clock, window: window}
}

type UseCase struct {
	shown     Messages
	board     Board
	publisher Publisher
	authors   Authors
	clock     cptime.Clock
	window    messages.Window
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

	shown, err := u.shown.Shown(ctx, in.MessageID, u.window.Since(u.clock.Now()), u.window.Size)
	if err != nil {
		return Out{}, fmt.Errorf("failed to read the message: %w", err)
	}
	if !shown {
		return Out{}, fmt.Errorf("%w: %q", reactions.ErrUnknownMessage, in.MessageID)
	}

	current, err := u.of(ctx, in.MessageID)
	if err != nil {
		return Out{}, err
	}
	if current.Given(in.Reaction, reactor) == in.On {
		return u.answer(ctx, current, reactor)
	}

	change := reactions.Change{MessageID: in.MessageID, Reaction: in.Reaction, Reactor: reactor, On: in.On, At: u.clock.Now()}
	if err := u.board.Save(ctx, change); err != nil {
		return Out{}, fmt.Errorf("failed to save the reaction: %w", err)
	}

	next, err := u.of(ctx, in.MessageID)
	if err != nil {
		return Out{}, err
	}
	tally := next.TallyOf(in.MessageID)
	named, err := u.authors.Authors(ctx, reactions.AccountsOf(tally.Counts))
	if err != nil {
		return Out{}, fmt.Errorf("failed to read who reacted: %w", err)
	}
	tally.Counts = reactions.Named(tally.Counts, named)
	u.publisher.Publish(feed.Update{Reactions: &tally})

	return Out{Counts: reactions.Named(next.Tally(reactor), named), Version: next.Version()}, nil
}

func (u *UseCase) answer(
	ctx context.Context,
	given reactions.Reactions,
	reactor reactions.Reactor,
) (Out, error) {
	counts := given.Tally(reactor)

	named, err := u.authors.Authors(ctx, reactions.AccountsOf(counts))
	if err != nil {
		return Out{}, fmt.Errorf("failed to read who reacted: %w", err)
	}
	return Out{Counts: reactions.Named(counts, named), Version: given.Version()}, nil
}

func (u *UseCase) of(ctx context.Context, id messages.MessageID) (reactions.Reactions, error) {
	given, err := u.board.Reactions(ctx, []messages.MessageID{id})
	if err != nil {
		return reactions.Reactions{}, fmt.Errorf("failed to read the reactions: %w", err)
	}
	return given[id], nil
}
