package uuid_id_provider

import (
	"fmt"

	"github.com/google/uuid"

	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes"
)

type Provider struct{}

var _ notes.IDProvider = Provider{}

func (Provider) NewID() (notes.NoteID, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return notes.NoteID{}, fmt.Errorf("failed to generate a uuidv7: %w", err)
	}
	return notes.NoteID(id), nil
}
