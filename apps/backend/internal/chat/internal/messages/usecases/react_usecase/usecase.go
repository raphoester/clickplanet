// Package react_usecase puts a reaction on a message, or takes it off, as whoever calls.
package react_usecase

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Board interface {
	React(ctx context.Context, change messages.ReactionChange) (messages.Reactions, error)
}

// Authors is the player module, asked who reacts: the same answer that names who posts.
type Authors interface {
	Author(ctx context.Context, account messages.AccountID, ip string) (messages.Author, error)
}

type In struct {
	// Account is the one the caller's click token names, or cpsession.NoAccount.
	Account   messages.AccountID
	MessageID messages.MessageID
	Reaction  messages.Reaction
	On        bool
}

func New(board Board, authors Authors, clock cptime.Clock) *UseCase {
	return &UseCase{board: board, authors: authors, clock: clock}
}

type UseCase struct {
	board   Board
	authors Authors
	clock   cptime.Clock
}

// Execute answers the message's reactions once the change landed, as the caller sees them.
func (u *UseCase) Execute(ctx context.Context, in In) ([]messages.Count, error) {
	author, err := u.authors.Author(ctx, in.Account, cpctx.GetSourceIP(ctx))
	if err != nil {
		return nil, fmt.Errorf("%w: %w", messages.ErrAuthorUnavailable, err)
	}

	reactor := messages.ReactorOf(in.Account, author)

	reactions, err := u.board.React(ctx, messages.ReactionChange{
		MessageID: in.MessageID,
		Reaction:  in.Reaction,
		Reactor:   reactor,
		On:        in.On,
		At:        u.clock.Now(),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to react: %w", err)
	}

	return reactions.Tally(reactor), nil
}
