package players

import (
	"context"
	"errors"
	"fmt"
)

// Who an account is, where the game shows it: its name, a guest's code, its admin crown and its color. Its badges, a
// streak and a title, are GetBadges', which the page reads beside it.
type Player struct {
	name  string
	guest bool
	admin bool
	color Color
}

func NamedPlayer(profile Profile) Player {
	return Player{name: DisplayNameOf(profile.name, ""), admin: profile.admin, color: profile.color}
}

func GuestPlayer(code GuestCode) Player {
	return Player{name: DisplayNameOf("", code), guest: true}
}

func (p Player) Name() string { return p.name }

func (p Player) Guest() bool { return p.guest }

func (p Player) Admin() bool { return p.admin }

func (p Player) Color() Color { return p.color }

func (p Player) Renamed(username Name) Player {
	p.name = DisplayNameOf(username, "")
	p.guest = false
	return p
}

type PlayerStore interface {
	Profile(ctx context.Context, account AccountID) (Profile, error)
	GuestCode(ctx context.Context, account AccountID) (GuestCode, error)
}

func PlayerOf(ctx context.Context, store PlayerStore, account AccountID) (Player, error) {
	profile, err := store.Profile(ctx, account)
	if err == nil {
		return NamedPlayer(profile), nil
	}
	if !errors.Is(err, ErrNoProfile) {
		return Player{}, fmt.Errorf("failed to read the profile: %w", err)
	}

	code, err := store.GuestCode(ctx, account)
	if err != nil {
		return Player{}, fmt.Errorf("failed to read the guest code: %w", err)
	}
	return GuestPlayer(code), nil
}
