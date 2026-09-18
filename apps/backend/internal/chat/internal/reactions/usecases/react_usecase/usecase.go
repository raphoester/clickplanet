// Package react_usecase puts a reaction on a message, or takes it off, as whoever calls.
package react_usecase

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/feed"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/reactions"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
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

// Authors is the player module, asked who reacts: the same answer that names who posts.
type Authors interface {
	Author(ctx context.Context, account messages.AccountID, ip string) (messages.Author, error)
}

// Publisher is the live feed: every change goes out as the message's whole tally.
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
	authors Authors,
	publisher Publisher,
	clock cptime.Clock,
	window messages.Window,
) *UseCase {
	return &UseCase{shown: shown, board: board, authors: authors, publisher: publisher, clock: clock, window: window}
}

type UseCase struct {
	shown     Messages
	board     Board
	authors   Authors
	publisher Publisher
	clock     cptime.Clock
	window    messages.Window

	// mu keeps a save, the read after it and the publish together, so the stream never sends an older tally last.
	mu sync.Mutex
}

// Execute answers the message's reactions as the caller sees them. A change that changes nothing is not saved.
func (u *UseCase) Execute(ctx context.Context, in In) ([]reactions.Count, error) {
	author, err := u.authors.Author(ctx, in.Account, cpctx.GetSourceIP(ctx))
	if err != nil {
		return nil, fmt.Errorf("%w: %w", messages.ErrAuthorUnavailable, err)
	}
	reactor := reactions.ReactorOf(in.Account, author)

	ctx, cancel := context.WithTimeout(ctx, writeTimeout)
	defer cancel()

	shown, err := u.shown.Shown(ctx, in.MessageID, u.window.Since(u.clock.Now()), u.window.Size)
	if err != nil {
		return nil, fmt.Errorf("failed to read the message: %w", err)
	}
	if !shown {
		return nil, fmt.Errorf("%w: %q", reactions.ErrUnknownMessage, in.MessageID)
	}

	u.mu.Lock()
	defer u.mu.Unlock()

	current, err := u.of(ctx, in.MessageID)
	if err != nil {
		return nil, err
	}
	if current.Given(in.Reaction, reactor) == in.On {
		return current.Tally(reactor), nil
	}

	change := reactions.Change{MessageID: in.MessageID, Reaction: in.Reaction, Reactor: reactor, On: in.On, At: u.clock.Now()}
	if err := u.board.Save(ctx, change); err != nil {
		return nil, fmt.Errorf("failed to save the reaction: %w", err)
	}

	next, err := u.of(ctx, in.MessageID)
	if err != nil {
		return nil, err
	}
	u.publisher.Publish(feed.Update{Reactions: &reactions.Tally{MessageID: in.MessageID, Counts: next.Tally(reactions.NoReactor)}})

	return next.Tally(reactor), nil
}

func (u *UseCase) of(ctx context.Context, id messages.MessageID) (reactions.Reactions, error) {
	given, err := u.board.Reactions(ctx, []messages.MessageID{id})
	if err != nil {
		return reactions.Reactions{}, fmt.Errorf("failed to read the reactions: %w", err)
	}
	return given[id], nil
}
