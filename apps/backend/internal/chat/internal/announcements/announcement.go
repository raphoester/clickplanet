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
	KindBomb    Kind = "bomb"
	KindMute    Kind = "mute"
	KindRound   Kind = "round"
	KindFortify Kind = "fortify"
)

var ErrUnknownKind = errors.New("an announcement of a kind nobody knows")

func Kinds() []Kind { return []Kind{KindBomb, KindMute, KindRound, KindFortify} }

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

// Below this a fortify is a small island somewhere, and the chat says nothing of it.
const fortifyNewsTiles = 50

type Fortify struct {
	country  string
	ground   string
	landmass uint32
	tiles    uint32
}

func FortifyOf(country string, ground string, landmass uint32, tiles uint32) Fortify {
	return Fortify{country: country, ground: ground, landmass: landmass, tiles: tiles}
}

func (f Fortify) Newsworthy() bool { return f.tiles >= fortifyNewsTiles }

type fortifyPayload struct {
	Country  string `json:"country"`
	Ground   string `json:"ground"`
	Landmass uint32 `json:"landmass"`
	Tiles    uint32 `json:"tiles"`
}

func (f Fortify) Payload() (json.RawMessage, error) {
	payload, err := json.Marshal(fortifyPayload{Country: f.country, Ground: f.ground, Landmass: f.landmass, Tiles: f.tiles})
	if err != nil {
		return nil, fmt.Errorf("failed to encode a fortify announcement: %w", err)
	}
	return payload, nil
}

type Muted struct {
	name     string
	duration time.Duration
}

func MutedOf(name string, duration time.Duration) Muted {
	return Muted{name: name, duration: duration}
}

type mutedPayload struct {
	Name    string `json:"name"`
	Seconds int64  `json:"seconds"`
}

func (m Muted) Payload() (json.RawMessage, error) {
	payload, err := json.Marshal(mutedPayload{Name: m.name, Seconds: int64(m.duration / time.Second)})
	if err != nil {
		return nil, fmt.Errorf("failed to encode a mute announcement: %w", err)
	}
	return payload, nil
}

const podium = 3

type Place struct {
	country string
	rank    uint32
	points  uint32
}

func PlaceOf(country string, rank uint32, points uint32) Place {
	return Place{country: country, rank: rank, points: points}
}

type Round struct {
	number uint32
	finale bool
	podium []Place
}

func RoundOf(number uint32, finale bool, places []Place) Round {
	kept := make([]Place, 0, podium)
	for _, place := range places {
		if place.rank <= podium && place.points > 0 {
			kept = append(kept, place)
		}
	}
	return Round{number: number, finale: finale, podium: kept}
}

type placePayload struct {
	Country string `json:"country"`
	Rank    uint32 `json:"rank"`
	Points  uint32 `json:"points"`
}

type roundPayload struct {
	Number uint32         `json:"number"`
	Finale bool           `json:"finale,omitempty"`
	Podium []placePayload `json:"podium"`
}

func (r Round) Payload() (json.RawMessage, error) {
	places := make([]placePayload, 0, len(r.podium))
	for _, place := range r.podium {
		places = append(places, placePayload{Country: place.country, Rank: place.rank, Points: place.points})
	}
	payload, err := json.Marshal(roundPayload{Number: r.number, Finale: r.finale, Podium: places})
	if err != nil {
		return nil, fmt.Errorf("failed to encode a round announcement: %w", err)
	}
	return payload, nil
}

type Storage interface {
	Append(ctx context.Context, announcement Announcement) error
	DeleteBefore(ctx context.Context, cutoff time.Time) (int64, error)
}
