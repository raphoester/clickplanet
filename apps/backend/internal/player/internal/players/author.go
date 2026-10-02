package players

import (
	"context"
	"errors"
	"fmt"
)

type Author struct {
	Name  string
	Guest bool
	Admin bool
}

type AuthorStore interface {
	Profile(ctx context.Context, account AccountID) (Profile, error)
	GuestCode(ctx context.Context, account AccountID) (GuestCode, error)
}

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
