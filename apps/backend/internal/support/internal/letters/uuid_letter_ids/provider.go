package uuid_letter_ids

import (
	"fmt"

	"github.com/google/uuid"

	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/letters"
)

type Provider struct{}

var _ letters.IDProvider = Provider{}

// NewID is a random UUID, so only the letter's own address names it.
func (Provider) NewID() (letters.LetterID, error) {
	id, err := uuid.NewRandom()
	if err != nil {
		return letters.LetterID{}, fmt.Errorf("failed to draw a letter id: %w", err)
	}
	return letters.LetterID(id), nil
}
