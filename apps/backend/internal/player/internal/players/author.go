package players

import (
	"context"
	"errors"
	"fmt"
)

// Author is who an account is to the other players: the name the game shows for it, and whether it is an admin.
type Author struct {
	// Name is the username, or ReservedPrefix and the guest code (DisplayNameOf). Never empty.
	Name string
	// Guest is an account that chose no username.
	Guest bool
	// Admin is false for a guest.
	Admin bool
}

// AuthorStore is what AuthorOf reads.
type AuthorStore interface {
	Profile(ctx context.Context, account AccountID) (Profile, error)
	GuestCode(ctx context.Context, account AccountID) (GuestCode, error)
}

// AuthorOf reads who the account is. The guest code is read only for an account with no username, and is
// ErrNoGuestCode until GuestCodes.Assign gave it one.
func AuthorOf(ctx context.Context, store AuthorStore, account AccountID) (Author, error) {
	profile, err := store.Profile(ctx, account)
	if err == nil {
		return Author{Name: DisplayNameOf(profile.Name, ""), Admin: profile.Admin}, nil
	}
	if !errors.Is(err, ErrNoProfile) {
		return Author{}, fmt.Errorf("failed to read the profile: %w", err)
	}

	code, err := store.GuestCode(ctx, account)
	if err != nil {
		return Author{}, fmt.Errorf("failed to read the guest code: %w", err)
	}
	return Author{Name: DisplayNameOf("", code), Guest: true}, nil
}
