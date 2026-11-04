package players

import (
	"context"
	"errors"
	"fmt"
)

// Who an account is, where the game shows it: its name, a guest's code, its admin crown and its color. Its badges, a
// streak and a title, are GetBadges', which the page reads beside it.
type Author struct {
	name  string
	guest bool
	admin bool
	color Color
}

func NamedAuthor(profile Profile) Author {
	return Author{name: DisplayNameOf(profile.name, ""), admin: profile.admin, color: profile.color}
}

func GuestAuthor(code GuestCode) Author {
	return Author{name: DisplayNameOf("", code), guest: true}
}

func (a Author) Name() string { return a.name }

func (a Author) Guest() bool { return a.guest }

func (a Author) Admin() bool { return a.admin }

func (a Author) Color() Color { return a.color }

func (a Author) Renamed(username Name) Author {
	a.name = DisplayNameOf(username, "")
	a.guest = false
	return a
}

type AuthorStore interface {
	Profile(ctx context.Context, account AccountID) (Profile, error)
	GuestCode(ctx context.Context, account AccountID) (GuestCode, error)
}

func AuthorOf(ctx context.Context, store AuthorStore, account AccountID) (Author, error) {
	profile, err := store.Profile(ctx, account)
	if err == nil {
		return NamedAuthor(profile), nil
	}
	if !errors.Is(err, ErrNoProfile) {
		return Author{}, fmt.Errorf("failed to read the profile: %w", err)
	}

	code, err := store.GuestCode(ctx, account)
	if err != nil {
		return Author{}, fmt.Errorf("failed to read the guest code: %w", err)
	}
	return GuestAuthor(code), nil
}
