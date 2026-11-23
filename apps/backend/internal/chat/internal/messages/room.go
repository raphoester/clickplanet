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

var (
	ErrNotInRoom  = errors.New("the caller does not play for the flag of the room")
	ErrRoomClosed = errors.New("the flag of the room is closed: its room is read, not written")
)

// Pledge is the flag an account plays for: no flag for an account that never pledged one, and closed once no player
// may play for it.
type Pledge struct {
	Flag   string
	Closed bool
}

type Flags interface {
	Flag(ctx context.Context, account AccountID) (Pledge, error)
}

// Only the accounts that play for a flag read and write in its room.
type Rooms struct {
	flags Flags
}

func NewRooms(flags Flags) Rooms {
	return Rooms{flags: flags}
}

func (r Rooms) EntryError(ctx context.Context, account AccountID, room Room) error {
	_, err := r.entered(ctx, account, room)
	return err
}

// SpeakError refuses a message or a reaction in the room of a flag closed, which its players still read.
func (r Rooms) SpeakError(ctx context.Context, account AccountID, room Room) error {
	pledge, err := r.entered(ctx, account, room)
	if err != nil {
		return err
	}
	if room != World && pledge.Closed {
		return fmt.Errorf("%w: %q", ErrRoomClosed, room)
	}
	return nil
}

func (r Rooms) entered(ctx context.Context, account AccountID, room Room) (Pledge, error) {
	if room == World {
		return Pledge{}, nil
	}
	if account == NoAccount {
		return Pledge{}, ErrNoAccount
	}

	pledge, err := r.flags.Flag(ctx, account)
	if err != nil {
		return Pledge{}, fmt.Errorf("failed to read the flag the caller plays for: %w", err)
	}
	if !room.Admits(pledge.Flag) {
		return Pledge{}, fmt.Errorf("%w: %q", ErrNotInRoom, room)
	}
	return pledge, nil
}
