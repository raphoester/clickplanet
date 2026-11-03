package messages

import (
	"context"
	"errors"
	"fmt"
)

// Where a message is said: the chat everyone reads, or the room of one flag.
type Room string

const World Room = "world"

func FlagRoom(flag string) Room {
	return Room(flag)
}

// No flag on the wire is the chat everyone reads.
func RoomOf(flagRoom string) Room {
	if flagRoom == "" {
		return World
	}
	return FlagRoom(flagRoom)
}

func (r Room) Admits(flag string) bool {
	return r == World || string(r) == flag
}

var ErrNotInRoom = errors.New("the caller does not play for the flag of the room")

type Flags interface {
	// Empty for an account that never pledged one.
	Flag(ctx context.Context, account AccountID) (string, error)
}

// Only the accounts that play for a flag read and write in its room.
type Rooms struct {
	flags Flags
}

func NewRooms(flags Flags) Rooms {
	return Rooms{flags: flags}
}

func (r Rooms) EntryError(ctx context.Context, account AccountID, room Room) error {
	if room == World {
		return nil
	}
	if account == NoAccount {
		return ErrNoAccount
	}

	flag, err := r.flags.Flag(ctx, account)
	if err != nil {
		return fmt.Errorf("failed to read the flag the caller plays for: %w", err)
	}
	if !room.Admits(flag) {
		return fmt.Errorf("%w: %q", ErrNotInRoom, room)
	}
	return nil
}
