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

func New(inner ledger.Storage, events Publisher) Storage {
	return Storage{Storage: inner, events: events}
}

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
