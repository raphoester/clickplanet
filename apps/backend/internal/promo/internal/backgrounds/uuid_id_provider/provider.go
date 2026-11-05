package uuid_id_provider

import (
	"fmt"

	"github.com/google/uuid"

	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/backgrounds"
)

type Provider struct{}

var _ backgrounds.IDProvider = Provider{}

// Random: the id is the file's name in the bucket and in its public URL, and the only thing an upload link is signed
// for.
func (Provider) NewID() (backgrounds.BackgroundID, error) {
	id, err := uuid.NewRandom()
	if err != nil {
		return backgrounds.BackgroundID{}, fmt.Errorf("failed to generate a uuid: %w", err)
	}
	return backgrounds.BackgroundID(id), nil
}
