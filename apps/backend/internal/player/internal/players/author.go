package players

import (
	"context"
	"errors"
	"fmt"
)

type Author struct {
	Name   string
	Guest  bool
	Admin  bool
	Color  Color
	Streak Streak
}

func (a Author) Shown(today Day) Author {
	if a.Guest {
		a.Streak = Streak{}
		return a
	}
	a.Streak = a.Streak.AsOf(today)
	return a
}

type AuthorStore interface {
	Profile(ctx context.Context, account AccountID) (Profile, error)
	GuestCode(ctx context.Context, account AccountID) (GuestCode, error)
	Stats(ctx context.Context, account AccountID) (Stats, error)
}

func AuthorOf(ctx context.Context, store AuthorStore, account AccountID) (Author, error) {
	author, err := namedAuthor(ctx, store, account)
	if err != nil {
		return Author{}, err
	}

	stats, err := store.Stats(ctx, account)
	if errors.Is(err, ErrNoStats) {
		return author, nil
	}
	if err != nil {
		return Author{}, fmt.Errorf("failed to read the stats: %w", err)
	}
	author.Streak = stats.Streak()
	return author, nil
}

func namedAuthor(ctx context.Context, store AuthorStore, account AccountID) (Author, error) {
	profile, err := store.Profile(ctx, account)
	if err == nil {
		return Author{Name: DisplayNameOf(profile.Name, ""), Admin: profile.Admin, Color: profile.Color}, nil
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
