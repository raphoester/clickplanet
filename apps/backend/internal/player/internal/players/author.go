package players

import (
	"context"
	"errors"
	"fmt"
)

type Author struct {
	name   string
	guest  bool
	admin  bool
	color  Color
	streak Streak
}

func NamedAuthor(profile Profile, streak Streak) Author {
	return Author{name: DisplayNameOf(profile.name, ""), admin: profile.admin, color: profile.color, streak: streak}
}

func GuestAuthor(code GuestCode, streak Streak) Author {
	return Author{name: DisplayNameOf("", code), guest: true, streak: streak}
}

func (a Author) Name() string { return a.name }

func (a Author) Guest() bool { return a.guest }

func (a Author) Admin() bool { return a.admin }

func (a Author) Color() Color { return a.color }

func (a Author) Streak() Streak { return a.streak }

func (a Author) Shown(today Day) Author {
	if a.guest {
		a.streak = Streak{}
		return a
	}
	a.streak = a.streak.AsOf(today)
	return a
}

func (a Author) Renamed(username Name) Author {
	a.name = DisplayNameOf(username, "")
	a.guest = false
	return a
}

type AuthorStore interface {
	Profile(ctx context.Context, account AccountID) (Profile, error)
	GuestCode(ctx context.Context, account AccountID) (GuestCode, error)
	Stats(ctx context.Context, account AccountID) (Stats, error)
}

func AuthorOf(ctx context.Context, store AuthorStore, account AccountID) (Author, error) {
	stats, err := store.Stats(ctx, account)
	switch {
	case errors.Is(err, ErrNoStats):
		stats = NewStats(account)
	case err != nil:
		return Author{}, fmt.Errorf("failed to read the stats: %w", err)
	}

	profile, err := store.Profile(ctx, account)
	if err == nil {
		return NamedAuthor(profile, stats.Streak()), nil
	}
	if !errors.Is(err, ErrNoProfile) {
		return Author{}, fmt.Errorf("failed to read the profile: %w", err)
	}

	code, err := store.GuestCode(ctx, account)
	if err != nil {
		return Author{}, fmt.Errorf("failed to read the guest code: %w", err)
	}
	return GuestAuthor(code, stats.Streak()), nil
}
