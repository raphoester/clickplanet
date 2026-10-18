package uuid_id_provider

import (
	"fmt"

	"github.com/google/uuid"

	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes"
)

type Provider struct{}

var (
	_ notes.IDProvider           = Provider{}
	_ notes.AttachmentIDProvider = Provider{}
)

func (Provider) NewID() (notes.NoteID, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return notes.NoteID{}, fmt.Errorf("failed to generate a uuidv7: %w", err)
	}
	return notes.NoteID(id), nil
}

// Random and not a v7: the id is the only proof that its holder uploaded the file.
func (Provider) NewAttachmentID() (notes.AttachmentID, error) {
	id, err := uuid.NewRandom()
	if err != nil {
		return notes.AttachmentID{}, fmt.Errorf("failed to generate a uuidv4: %w", err)
	}
	return notes.AttachmentID(id), nil
}
