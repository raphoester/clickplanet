// Package get_history_usecase reads the recent messages a joining client is shown.
package get_history_usecase

import (
	"context"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
)

type HistoryReader interface {
	History(ctx context.Context, viewer messages.Reactor) []messages.Message
}

// Authors is the player module, asked who reads, so the caller's own reactions can say so.
type Authors interface {
	Author(ctx context.Context, account messages.AccountID, ip string) (messages.Author, error)
}

func New(reader HistoryReader, authors Authors) *UseCase {
	return &UseCase{reader: reader, authors: authors}
}

type UseCase struct {
	reader  HistoryReader
	authors Authors
}

// Execute serves the history even when the player module does not answer: nothing is then marked as the caller's.
func (u *UseCase) Execute(ctx context.Context, account messages.AccountID) []messages.Message {
	viewer := messages.NoReactor
	if author, err := u.authors.Author(ctx, account, cpctx.GetSourceIP(ctx)); err == nil {
		viewer = messages.ReactorOf(account, author)
	}

	return u.reader.History(ctx, viewer)
}
