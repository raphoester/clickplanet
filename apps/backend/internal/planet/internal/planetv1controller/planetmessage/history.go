package planetmessage

import (
	"cmp"
	"maps"
	"slices"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger"
)

func MapHistory(frames []ledger.Frame, hits map[uint32]uint32) *planetv1.MapHistory {
	answer := &planetv1.MapHistory{}
	for i, frame := range frames {
		shown := &planetv1.MapFrame{At: timestamppb.New(frame.At), Owners: make([]*planetv1.TileOwner, 0, len(frame.Owners))}
		for _, tile := range slices.Sorted(maps.Keys(frame.Owners)) {
			owner := frame.Owners[tile]
			shown.Owners = append(shown.Owners, &planetv1.TileOwner{
				Tile: tile, CountryId: owner.Country, SecondsIntoStep: secondsInto(frames, i, owner),
			})
		}
		answer.Frames = append(answer.Frames, shown)
	}
	for tile, hits := range hits {
		answer.Hits = append(answer.Hits, &planetv1.TileHits{Tile: tile, Hits: hits})
	}
	slices.SortFunc(answer.Hits, func(a, b *planetv1.TileHits) int { return cmp.Compare(a.GetTile(), b.GetTile()) })
	return answer
}

// The first frame has no step before it.
func secondsInto(frames []ledger.Frame, frame int, owner ledger.Owner) uint32 {
	if frame == 0 || owner.At.Before(frames[frame-1].At) {
		return 0
	}
	return uint32(owner.At.Sub(frames[frame-1].At) / time.Second)
}
