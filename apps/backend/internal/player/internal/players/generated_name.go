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

// Assign answers whether it made the profile: false for an account that had one.
func (g GeneratedNames) Assign(ctx context.Context, account AccountID, at time.Time) (bool, error) {
	for range maxDraws {
		name, err := g.generator.NewName()
		if err != nil {
			return false, fmt.Errorf("failed to draw a name: %w", err)
		}

		err = g.store.CreateProfile(ctx, NewProfile(account, name, at))
		if errors.Is(err, ErrProfileExists) {
			return false, nil
		}
		if errors.Is(err, ErrNameTaken) {
			continue
		}
		if err != nil {
			return false, fmt.Errorf("failed to save the name: %w", err)
		}
		return true, nil
	}
	return false, ErrNoFreeName
}
