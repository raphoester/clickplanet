// Package publishing_ledger_storage tells the other modules of every take an account made, beside the
// ledger that records it: planet.v1.TileTaken, one event per tile.
package publishing_ledger_storage

import (
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger"
)

// Publisher is the event bus. Publish never blocks, so a click never waits on a listener.
type Publisher interface {
	Publish(event proto.Message)
}

func New(inner ledger.Storage, events Publisher) Storage {
	return Storage{Storage: inner, events: events}
}

// Storage is the ledger, and every take it appends with an account is published once it is recorded.
type Storage struct {
	ledger.Storage
	events Publisher
}

var _ ledger.Storage = Storage{}

func (s Storage) Append(taking ledger.Taking) {
	s.Storage.Append(taking)

	if taking.Account == "" {
		return
	}

	s.events.Publish(&planetv1.TileTaken{
		AccountId: taking.Account,
		TileId:    taking.Tile,
		Country:   taking.Country,
		TakenAt:   timestamppb.New(taking.At),
	})
}
