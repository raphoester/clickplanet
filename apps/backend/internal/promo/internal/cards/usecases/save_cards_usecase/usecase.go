package save_cards_usecase

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/cards"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
)

type Sets interface {
	Names(ctx context.Context) ([]string, error)
}

type Storage interface {
	Save(ctx context.Context, deck cards.Deck) error
}

const writeTimeout = 10 * time.Second

func New(sets Sets, storage Storage, transactioner cpatomic.Transactioner) *UseCase {
	return &UseCase{sets: sets, storage: storage, transactioner: transactioner}
}

type UseCase struct {
	sets          Sets
	storage       Storage
	transactioner cpatomic.Transactioner
}

// A set planet does not keep shows nothing: its cards would never be read.
func (u *UseCase) Execute(ctx context.Context, set cards.SetName, drafts []cards.Draft) error {
	ctx, cancel := context.WithTimeout(ctx, writeTimeout)
	defer cancel()

	deck, err := cards.NewDeck(set, drafts)
	if err != nil {
		return fmt.Errorf("cannot keep the cards: %w", err)
	}
	names, err := u.sets.Names(ctx)
	if err != nil {
		return fmt.Errorf("failed to read the rule sets: %w", err)
	}
	if !slices.Contains(names, string(set)) {
		return fmt.Errorf("%w: %q", cards.ErrUnknownSet, set)
	}
	if err := u.transactioner.InTx(ctx, func(ctx context.Context) error {
		return u.storage.Save(ctx, deck)
	}); err != nil {
		return fmt.Errorf("failed to keep the cards of %s: %w", set, err)
	}
	return nil
}
