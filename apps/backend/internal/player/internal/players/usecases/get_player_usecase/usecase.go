package get_player_usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/titles"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Profiles interface {
	ProfileNamed(ctx context.Context, name players.Name) (players.Profile, error)
}

type Stats interface {
	Stats(ctx context.Context, account players.AccountID) (players.Stats, error)
}

type Titles interface {
	TitlesOf(ctx context.Context, account players.AccountID) ([]titles.Title, error)
}

type Player struct {
	players.Player
	Titles []titles.Title
}

type Accounts interface {
	CreatedAt(ctx context.Context, account players.AccountID) (time.Time, error)
}

type UseCase struct {
	profiles Profiles
	stats    Stats
	titles   Titles
	accounts Accounts
	clock    cptime.Clock
}

func New(profiles Profiles, stats Stats, titles Titles, accounts Accounts, clock cptime.Clock) *UseCase {
	return &UseCase{profiles: profiles, stats: stats, titles: titles, accounts: accounts, clock: clock}
}

func (u *UseCase) Execute(ctx context.Context, value string) (Player, error) {
	name, err := players.NameOf(value)
	if err != nil {
		return Player{}, fmt.Errorf("%w: %w", players.ErrNoProfile, err)
	}

	profile, err := u.profiles.ProfileNamed(ctx, name)
	if errors.Is(err, players.ErrNoProfile) {
		return Player{}, err //nolint:wrapcheck // the handler maps the port's sentinel.
	}
	if err != nil {
		return Player{}, fmt.Errorf("failed to read the profile: %w", err)
	}

	stats, err := u.stats.Stats(ctx, profile.Account)
	switch {
	case errors.Is(err, players.ErrNoStats):
		stats = players.Stats{Account: profile.Account}
	case err != nil:
		return Player{}, fmt.Errorf("failed to read the stats: %w", err)
	}

	held, err := u.titles.TitlesOf(ctx, profile.Account)
	if err != nil {
		return Player{}, fmt.Errorf("failed to read the titles: %w", err)
	}

	createdAt, err := u.accounts.CreatedAt(ctx, profile.Account)
	if err != nil {
		return Player{}, fmt.Errorf("failed to ask when the account was made: %w", err)
	}

	return Player{
		Player: players.Player{
			Name:      profile.Name,
			Stats:     stats.AsOf(players.DayOf(u.clock.Now())),
			CreatedAt: createdAt,
			Admin:     profile.Admin,
			Color:     profile.Color,
		},
		Titles: held,
	}, nil
}
