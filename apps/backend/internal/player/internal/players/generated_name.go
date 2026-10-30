package players

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"golang.org/x/text/unicode/norm"
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
func (g GeneratedNames) Assign(ctx context.Context, account AccountID, suggested string, at time.Time) (bool, error) {
	if name, err := suggestedNameOf(suggested); err == nil {
		created, err := g.create(ctx, account, name, at)
		if !errors.Is(err, ErrNameTaken) {
			return created, err
		}
	}

	for range maxDraws {
		name, err := g.generator.NewName()
		if err != nil {
			return false, fmt.Errorf("failed to draw a name: %w", err)
		}

		created, err := g.create(ctx, account, name, at)
		if errors.Is(err, ErrNameTaken) {
			continue
		}
		return created, err
	}
	return false, ErrNoFreeName
}

func (g GeneratedNames) create(ctx context.Context, account AccountID, name Name, at time.Time) (bool, error) {
	err := g.store.CreateProfile(ctx, NewProfile(account, name, at))
	if errors.Is(err, ErrProfileExists) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to save the name: %w", err)
	}
	return true, nil
}

// CrazyGames' usernames hold dots and run to 20 characters.
func suggestedNameOf(value string) (Name, error) {
	value = strings.ReplaceAll(norm.NFC.String(value), ".", "_")
	if runes := []rune(value); len(runes) > MaxNameLength {
		value = string(runes[:MaxNameLength])
	}
	return NameOf(value)
}
