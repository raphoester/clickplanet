package publishing_ledger_storage

import (
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger"
)

type Publisher interface {
	Publish(event proto.Message)
}

type Grounds interface {
	CountryOf(tile uint32) string
}

func New(inner ledger.Storage, grounds Grounds, events Publisher) Storage {
	return Storage{Storage: inner, grounds: grounds, events: events}
}

type Storage struct {
	ledger.Storage
	grounds Grounds
	events  Publisher
}

var _ ledger.Storage = Storage{}

func (s Storage) Append(event ledger.Event) {
	s.Storage.Append(event)

	event.Replay(func(taking ledger.Taking) {
		if taking.Account == "" || taking.Cleared() {
			return
		}

		s.events.Publish(&planetv1.TileTaken{
			AccountId: taking.Account,
			TileId:    taking.Tile,
			Country:   taking.Country,
			TakenAt:   timestamppb.New(taking.At),
			Previous:  taking.Previous,
			Ground:    s.grounds.CountryOf(taking.Tile),
		})
	})
}
