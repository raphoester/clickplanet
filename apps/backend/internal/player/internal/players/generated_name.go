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

type ProfileCreator interface {
	CreateProfile(ctx context.Context, profile Profile) error
}

var ErrNoFreeName = errors.New("every name drawn was taken")

type GeneratedNames struct {
	store     ProfileCreator
	generator NameGenerator
}

func NewGeneratedNames(store ProfileCreator, generator NameGenerator) GeneratedNames {
	return GeneratedNames{store: store, generator: generator}
}

func (g GeneratedNames) Assign(ctx context.Context, account AccountID, at time.Time) error {
	for range maxDraws {
		name, err := g.generator.NewName()
		if err != nil {
			return fmt.Errorf("failed to draw a name: %w", err)
		}

		err = g.store.CreateProfile(ctx, NewProfile(account, name, at))
		if errors.Is(err, ErrProfileExists) {
			return nil
		}
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
		_, hasName := named[stats.account]
		if known[stats.account].linked && !hasName {
			nameless = append(nameless, stats.account)
		}
	}
	return nameless
}

func AccountsOf(page []Stats) []AccountID {
	accounts := make([]AccountID, len(page))
	for i, stats := range page {
		accounts[i] = stats.account
	}
	return accounts
}
