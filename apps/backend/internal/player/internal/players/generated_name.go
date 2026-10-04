package players

import (
	"context"
	"errors"
	"fmt"
	"time"
)

type NameGenerator interface {
	NewName() (Name, error)
}

type ProfileStore interface {
	Profile(ctx context.Context, account AccountID) (Profile, error)
	SaveProfile(ctx context.Context, profile Profile) error
}

var ErrNoFreeName = errors.New("every name drawn was taken")

type GeneratedNames struct {
	store     ProfileStore
	generator NameGenerator
}

func NewGeneratedNames(store ProfileStore, generator NameGenerator) GeneratedNames {
	return GeneratedNames{store: store, generator: generator}
}

func (g GeneratedNames) Assign(ctx context.Context, account AccountID, at time.Time) error {
	_, err := g.store.Profile(ctx, account)
	if err == nil {
		return nil
	}
	if !errors.Is(err, ErrNoProfile) {
		return fmt.Errorf("failed to read the profile: %w", err)
	}

	for range maxDraws {
		name, err := g.generator.NewName()
		if err != nil {
			return fmt.Errorf("failed to draw a name: %w", err)
		}

		err = g.store.SaveProfile(ctx, Profile{Account: account, Name: name, UpdatedAt: at})
		if errors.Is(err, ErrNameTaken) {
			continue
		}
		if err != nil {
			return fmt.Errorf("failed to save the name: %w", err)
		}
		return nil
	}
	return ErrNoFreeName
}

func NamelessLinked(page []Stats, known map[AccountID]Account, named map[AccountID]Name) []AccountID {
	var nameless []AccountID
	for _, stats := range page {
		_, hasName := named[stats.Account]
		if known[stats.Account].Linked && !hasName {
			nameless = append(nameless, stats.Account)
		}
	}
	return nameless
}

func AccountsOf(page []Stats) []AccountID {
	accounts := make([]AccountID, len(page))
	for i, stats := range page {
		accounts[i] = stats.Account
	}
	return accounts
}
