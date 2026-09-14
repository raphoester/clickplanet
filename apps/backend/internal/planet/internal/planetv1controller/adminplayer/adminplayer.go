package adminplayer

import (
	"time"

	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger"
)

func Encode(players []ledger.Player) []*planetv1.Player {
	encoded := make([]*planetv1.Player, 0, len(players))
	for _, player := range players {
		encoded = append(encoded, &planetv1.Player{
			Scope:          player.Scope,
			Tiles:          uint32(player.Tiles), //nolint:gosec // a tile count, bounded by the map.
			FirstAt:        timestamppb.New(player.FirstAt),
			LastAt:         timestamppb.New(player.LastAt),
			Banned:         player.Banned,
			BannedUntil:    timestampOrNil(player.BannedUntil),
			Offence:        uint32(player.Offence), //nolint:gosec // an offence count, never negative.
			ActiveFor:      durationpb.New(player.ActiveFor()),
			TilesPerMinute: player.TilesPerMinute(),
		})
	}

	return encoded
}

func timestampOrNil(at time.Time) *timestamppb.Timestamp {
	if at.IsZero() {
		return nil
	}
	return timestamppb.New(at)
}
