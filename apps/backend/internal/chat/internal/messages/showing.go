package messages

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

// What an account is shown: a message among the newest of its room, in a room the account may enter.
type Showing struct {
	shown  Shown
	rooms  Rooms
	clock  cptime.Clock
	window Window
}

func NewShowing(shown Shown, rooms Rooms, clock cptime.Clock, window Window) Showing {
	return Showing{shown: shown, rooms: rooms, clock: clock, window: window}
}

func (s Showing) Message(ctx context.Context, account AccountID, id MessageID) (Message, error) {
	message, err := s.shown.Message(ctx, id, s.window.Since(s.clock.Now()), s.window.Size())
	if err != nil {
		return Message{}, fmt.Errorf("failed to read the message: %w", err)
	}
	if err := s.rooms.EntryError(ctx, account, message.room); err != nil {
		return Message{}, err
	}
	return message, nil
}
