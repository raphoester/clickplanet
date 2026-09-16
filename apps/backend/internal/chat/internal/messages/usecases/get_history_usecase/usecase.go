// Package get_history_usecase reads the recent messages a joining client is shown.
package get_history_usecase

import (
	"context"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
)

type HistoryReader interface {
	History(ctx context.Context) []messages.Message
}

type BanChecker interface {
	Banned(tag string) bool
}

func New(reader HistoryReader, bans BanChecker) *UseCase {
	return &UseCase{reader: reader, bans: bans}
}

type UseCase struct {
	reader HistoryReader
	bans   BanChecker
}

// Execute blanks a banned author here rather than when the ban is passed, so the
// stored text survives the ban and a lifted one needs nothing put back.
func (u *UseCase) Execute(ctx context.Context) []messages.Message {
	history := u.reader.History(ctx)

	for i, message := range history {
		if u.bans.Banned(message.AuthorTag) {
			history[i] = message.Redact()
		}
	}

	return history
}
