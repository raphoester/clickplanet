package announcements

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type AnnouncementID uuid.UUID

type Kind string

const KindBomb Kind = "bomb"

func (k Kind) Known() bool { return k == KindBomb }

type Announcement struct {
	ID      AnnouncementID
	Kind    Kind
	At      time.Time
	Payload json.RawMessage
}

type Bomb struct {
	Country string `json:"country"`
	Ground  string `json:"ground,omitempty"`
	Tile    uint32 `json:"tile,omitempty"`
	Cleared uint32 `json:"cleared"`
}

func (b Bomb) Payload() (json.RawMessage, error) {
	payload, err := json.Marshal(b)
	if err != nil {
		return nil, fmt.Errorf("failed to encode a bomb announcement: %w", err)
	}
	return payload, nil
}

type Storage interface {
	Append(ctx context.Context, announcement Announcement) error
	DeleteBefore(ctx context.Context, cutoff time.Time) (int64, error)
}
