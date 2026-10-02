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

// Authors is the player module, asked who the people under a message's reactions are. The chat keeps accounts,
// never names, so this is where a reaction gets one — on the way out, for the answer and the broadcast alike.
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
	// One ask names everybody under the message, for the answer and for the frame that goes to every other
	// client: neither is worth a second one, and a reader of the stream has no way to ask for itself.
	tally := next.TallyOf(in.MessageID)
	named, err := u.authors.Authors(ctx, reactions.AccountsOf(tally.Counts))
	if err != nil {
		return Out{}, fmt.Errorf("failed to read who reacted: %w", err)
	}
	tally.Counts = reactions.Named(tally.Counts, named)
	u.publisher.Publish(feed.Update{Reactions: &tally})

	return Out{Counts: reactions.Named(next.Tally(reactor), named), Version: next.Version()}, nil
}

// answer is what a change that changed nothing says: what is already there, named. It costs the same one ask,
// since the caller is shown the same list either way.
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
