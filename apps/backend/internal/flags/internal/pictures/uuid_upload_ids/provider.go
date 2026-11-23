package uuid_upload_ids

import (
	"fmt"

	"github.com/google/uuid"

	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/pictures"
)

type Provider struct{}

var _ pictures.IDProvider = Provider{}

func (Provider) NewID() (pictures.UploadID, error) {
	id, err := uuid.NewRandom()
	if err != nil {
		return pictures.UploadID{}, fmt.Errorf("failed to draw an upload id: %w", err)
	}
	return pictures.UploadID(id), nil
}
