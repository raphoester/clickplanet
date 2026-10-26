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
		refused := !player.RefusedUntil.IsZero()
		encoded = append(encoded, &planetv1.Player{
			Scope:          player.Scope,
			AccountId:      player.Account,
			FlagCountryId:  player.Flag,
			Tiles:          uint32(player.Tiles), //nolint:gosec // a tile count, bounded by the map.
			FirstAt:        timestamppb.New(player.FirstAt),
			LastAt:         timestamppb.New(player.LastAt),
			Banned:         &refused,
			BannedUntil:    timestampOrNil(player.RefusedUntil),
			ActiveFor:      durationpb.New(player.ActiveFor()),
			TilesPerMinute: player.TilesPerMinute(),
			Takes:          uint32(player.Takes), //nolint:gosec // a take count, bounded by the ledger's cap.
			TakesPerMinute: player.TakesPerMinute(),
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
