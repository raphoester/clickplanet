package uuid_id_provider

import (
	"fmt"

	"github.com/google/uuid"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/videos"
)

type Provider struct{}

var _ videos.IDProvider = Provider{}

// Random: the id is the file's name in the bucket, and the only thing an upload link is signed for.
func (Provider) NewID() (videos.VideoID, error) {
	id, err := uuid.NewRandom()
	if err != nil {
		return videos.VideoID{}, fmt.Errorf("failed to generate a uuid: %w", err)
	}
	return videos.VideoID(id), nil
}
