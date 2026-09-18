// Package announcements is what the chat says on its own: a line between the messages, with no sender.
package announcements

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// AnnouncementID names an announcement. The server makes it when it keeps the announcement.
type AnnouncementID uuid.UUID

// Kind is what happened, and so how to read an announcement's payload.
type Kind string

// KindBomb is a bomb that landed. Its payload is a Bomb.
const KindBomb Kind = "bomb"

// Announcement has no sender, so nothing in it is personal data. Payload is the values its kind's line is
// written from, as a JSON object: the client writes the line, so a new wording needs no migration.
type Announcement struct {
	ID      AnnouncementID
	Kind    Kind
	At      time.Time
	Payload json.RawMessage
}

// Bomb is the payload of a KindBomb announcement.
type Bomb struct {
	// Country is the code of the country the bomber played for.
	Country string `json:"country"`
	// Ground is the code of the country whose ground it hit: empty in the sea, and on no country's ground.
	Ground string `json:"ground,omitempty"`
	// Tile is the tile it hit, 0 in the sea.
	Tile uint32 `json:"tile,omitempty"`
	// Cleared is how many held tiles it cleared.
	Cleared uint32 `json:"cleared"`
}

// Payload is the bomb as an announcement's payload.
func (b Bomb) Payload() (json.RawMessage, error) {
	payload, err := json.Marshal(b)
	if err != nil {
		return nil, fmt.Errorf("failed to encode a bomb announcement: %w", err)
	}
	return payload, nil
}

// Storage is where announcements are kept. StorageContractSuite pins what every adapter does.
type Storage interface {
	Append(ctx context.Context, announcement Announcement) error
	// Recent is the newest limit announcements made at or after since, oldest first.
	Recent(ctx context.Context, since time.Time, limit int) ([]Announcement, error)
	// DeleteBefore removes every announcement made before cutoff and says how many.
	DeleteBefore(ctx context.Context, cutoff time.Time) (int64, error)
}
