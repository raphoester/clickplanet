package postgres_card_store

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/backgrounds"
	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/cards"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

const foreignKeyViolation = "23503"

func New(db cppg.Conn) *Store {
	return &Store{db: db}
}

type Store struct {
	db cppg.Conn
}

// Save replaces the set's cards: call it in a transaction. A card on a background the library does not keep is
// refused by the foreign key.
func (s *Store) Save(ctx context.Context, deck cards.Deck) error {
	kept := deck.TakeSnapshot()
	if _, err := s.db.On(ctx).ExecContext(ctx, `DELETE FROM cards WHERE rule_set = $1`, string(kept.Set)); err != nil {
		return fmt.Errorf("failed to clear the cards of %s: %w", kept.Set, err)
	}
	for position, card := range kept.Cards {
		_, err := s.db.On(ctx).ExecContext(ctx, `
			INSERT INTO cards (rule_set, position, background_id, headline, line) VALUES ($1, $2, $3, $4, $5)
		`, string(kept.Set), position, uuid.UUID(card.Background), card.Headline, card.Line)
		var pqErr *pq.Error
		switch {
		case errors.As(err, &pqErr) && pqErr.Code == foreignKeyViolation:
			return fmt.Errorf("%w: %s, on card %d of %s", backgrounds.ErrNoBackground, card.Background, position+1, kept.Set)
		case err != nil:
			return fmt.Errorf("failed to keep card %d of %s: %w", position+1, kept.Set, err)
		}
	}
	return nil
}
