package announcements

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/google/uuid"
)

type AnnouncementID uuid.UUID

type Kind string

const (
	KindBomb        Kind = "bomb"
	KindLeadChanged Kind = "lead_changed"
	KindSeasonWon   Kind = "season_won"
)

var (
	ErrUnknownKind = errors.New("an announcement of a kind nobody knows")

	ErrKept = errors.New("an announcement with this id is already kept")
)

func Kinds() []Kind { return []Kind{KindBomb, KindLeadChanged, KindSeasonWon} }

func (k Kind) Known() bool { return slices.Contains(Kinds(), k) }

type Announcement struct {
	id      AnnouncementID
	kind    Kind
	at      time.Time
	payload json.RawMessage
}

func NewAnnouncement(id AnnouncementID, kind Kind, at time.Time, payload json.RawMessage) Announcement {
	return Announcement{id: id, kind: kind, at: at, payload: payload}
}

func (a Announcement) ID() AnnouncementID { return a.id }

func (a Announcement) Kind() Kind { return a.kind }

func (a Announcement) At() time.Time { return a.at }

func (a Announcement) Payload() json.RawMessage { return a.payload }

type Bomb struct {
	country string
	ground  string
	tile    uint32
	cleared uint32
}

func BombOf(country string, ground string, tile uint32, cleared uint32) Bomb {
	return Bomb{country: country, ground: ground, tile: tile, cleared: cleared}
}

type bombPayload struct {
	Country string `json:"country"`
	Ground  string `json:"ground,omitempty"`
	Tile    uint32 `json:"tile,omitempty"`
	Cleared uint32 `json:"cleared"`
}

func (b Bomb) Payload() (json.RawMessage, error) {
	payload, err := json.Marshal(bombPayload{Country: b.country, Ground: b.ground, Tile: b.tile, Cleared: b.cleared})
	if err != nil {
		return nil, fmt.Errorf("failed to encode a bomb announcement: %w", err)
	}
	return payload, nil
}

type LeadChange struct {
	season uint32
	leader string
	passed string
}

func LeadChangeOf(season uint32, leader string, passed string) LeadChange {
	return LeadChange{season: season, leader: leader, passed: passed}
}

type leadChangePayload struct {
	Season uint32 `json:"season"`
	Leader string `json:"leader"`
	Passed string `json:"passed"`
}

func (l LeadChange) Payload() (json.RawMessage, error) {
	payload, err := json.Marshal(leadChangePayload{Season: l.season, Leader: l.leader, Passed: l.passed})
	if err != nil {
		return nil, fmt.Errorf("failed to encode a lead change announcement: %w", err)
	}
	return payload, nil
}

type Win struct {
	season uint32
	winner string
}

func WinOf(season uint32, winner string) Win {
	return Win{season: season, winner: winner}
}

type winPayload struct {
	Season uint32 `json:"season"`
	Winner string `json:"winner"`
}

func (w Win) Payload() (json.RawMessage, error) {
	payload, err := json.Marshal(winPayload{Season: w.season, Winner: w.winner})
	if err != nil {
		return nil, fmt.Errorf("failed to encode a season won announcement: %w", err)
	}
	return payload, nil
}

type Storage interface {
	Append(ctx context.Context, announcement Announcement) error
	DeleteBefore(ctx context.Context, cutoff time.Time) (int64, error)
}
