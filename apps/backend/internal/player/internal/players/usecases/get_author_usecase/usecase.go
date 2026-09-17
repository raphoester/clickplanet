// Package get_author_usecase says who a caller is, for another module: its username and its tag.
package get_author_usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
)

type Profiles interface {
	Profile(ctx context.Context, account players.AccountID) (players.Profile, error)
}

type In struct {
	// Account is cpsession.NoAccount for a caller with none.
	Account players.AccountID
	IP      string
}

type UseCase struct {
	profiles Profiles
	tagSalt  string
}

func New(profiles Profiles, tagSalt string) *UseCase {
	return &UseCase{profiles: profiles, tagSalt: tagSalt}
}

// Execute answers no name for a caller with no account, or an account that never chose one.
func (u *UseCase) Execute(ctx context.Context, in In) (players.Author, error) {
	author := players.Author{Tag: players.TagOf(u.tagSalt, in.IP)}
	if in.Account == cpsession.NoAccount {
		return author, nil
	}

	profile, err := u.profiles.Profile(ctx, in.Account)
	if errors.Is(err, players.ErrNoProfile) {
		return author, nil
	}
	if err != nil {
		return players.Author{}, fmt.Errorf("failed to read the profile: %w", err)
	}

	author.Name = profile.Name
	return author, nil
}
