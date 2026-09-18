// Package react_usecase puts a reaction on a message, or takes it off, as whoever calls.
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

// Messages says whether a message is one the chat shows: nobody can react to any other.
type Messages interface {
	Shown(ctx context.Context, id messages.MessageID, since time.Time, limit int) (bool, error)
}

type Board interface {
	Save(ctx context.Context, change reactions.Change) error
	Reactions(ctx context.Context, ids []messages.MessageID) (map[messages.MessageID]reactions.Reactions, error)
}

// Publisher is the live feed: every change goes out as the message's whole tally, versioned, so the order the
// tallies are published in does not matter.
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

func New(
	shown Messages,
	board Board,
	publisher Publisher,
	clock cptime.Clock,
	window messages.Window,
) *UseCase {
	return &UseCase{shown: shown, board: board, publisher: publisher, clock: clock, window: window}
}

type UseCase struct {
	shown     Messages
	board     Board
	publisher Publisher
	clock     cptime.Clock
	window    messages.Window
}

// Out is the message's reactions as the caller sees them, and their version.
type Out struct {
	Counts  []reactions.Count
	Version uint64
}

// Execute answers the message's reactions once the change landed. A change that changes nothing is not saved.
// A caller with no account is refused: a reaction is an account's.
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
		return outOf(current, reactor), nil
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
	u.publisher.Publish(feed.Update{Reactions: &tally})

	return outOf(next, reactor), nil
}

func outOf(given reactions.Reactions, reactor reactions.Reactor) Out {
	return Out{Counts: given.Tally(reactor), Version: given.Version()}
}

func (u *UseCase) of(ctx context.Context, id messages.MessageID) (reactions.Reactions, error) {
	given, err := u.board.Reactions(ctx, []messages.MessageID{id})
	if err != nil {
		return reactions.Reactions{}, fmt.Errorf("failed to read the reactions: %w", err)
	}
	return given[id], nil
}
