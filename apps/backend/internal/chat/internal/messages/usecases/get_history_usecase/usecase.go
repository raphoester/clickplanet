// Package get_history_usecase reads the recent messages a joining client is shown.
package get_history_usecase

import (
	"context"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
)

type HistoryReader interface {
	History(ctx context.Context) []messages.Message
}

func New(reader HistoryReader) *UseCase {
	return &UseCase{reader: reader}
}

type UseCase struct {
	reader HistoryReader
}

func (u *UseCase) Execute(ctx context.Context) []messages.Message {
	return u.reader.History(ctx)
}
