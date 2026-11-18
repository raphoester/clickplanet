package planetmessage

import (
	"cmp"
	"encoding/binary"
	"maps"
	"math"
	"slices"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/eras"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger"
)

func MapHistory(frames []ledger.Frame, hits map[ledger.EraTile]uint32, covering []eras.Era) *planetv1.MapHistory {
	answer := &planetv1.MapHistory{Eras: Eras(covering)}
	for i, frame := range frames {
		if frame.Whole {
			answer.Frames = append(answer.Frames, &planetv1.MapFrame{At: timestamppb.New(frame.At), Whole: Whole(frame.Owners)})
			continue
		}
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
		answer.Hits = append(answer.Hits, &planetv1.TileHits{Tile: tile.Tile, Hits: hits, Era: uint32(tile.Era)}) //nolint:gosec // a few eras.
	}
	slices.SortFunc(answer.Hits, func(a, b *planetv1.TileHits) int {
		return cmp.Or(cmp.Compare(a.GetEra(), b.GetEra()), cmp.Compare(a.GetTile(), b.GetTile()))
	})
	return answer
}

// Whole writes every tile held as GetMap does: two bytes a tile from the first held, into the flags sorted after "".
func Whole(owners map[uint32]ledger.Owner) *planetv1.WholeMap {
	held := map[string]uint16{}
	first, last := uint32(math.MaxUint32), uint32(0)
	for tile, owner := range owners {
		if owner.Country != "" {
			held[owner.Country] = 0
			first, last = min(first, tile), max(last, tile)
		}
	}
	codes := append([]string{""}, slices.Sorted(maps.Keys(held))...)
	if len(held) == 0 {
		return &planetv1.WholeMap{Codes: codes}
	}
	for i, country := range codes[1:] {
		held[country] = uint16(i + 1) //nolint:gosec // a few hundred flags.
	}
	tiles := make([]byte, 2*(last-first+1))
	for tile, owner := range owners {
		if owner.Country != "" {
			binary.LittleEndian.PutUint16(tiles[2*(tile-first):], held[owner.Country])
		}
	}
	return &planetv1.WholeMap{StartTileId: first, Codes: codes, Tiles: tiles}
}

// Eras names each era's board by its name, oldest first.
func Eras(covering []eras.Era) []*planetv1.BoardEra {
	shown := make([]*planetv1.BoardEra, len(covering))
	for i, era := range covering {
		shown[i] = &planetv1.BoardEra{StartedAt: timestamppb.New(era.StartedAt), Board: string(era.Board.Name)}
	}
	return shown
}

// The first frame has no step before it.
func secondsInto(frames []ledger.Frame, frame int, owner ledger.Owner) uint32 {
	if frame == 0 || owner.At.Before(frames[frame-1].At) {
		return 0
	}
	return uint32(owner.At.Sub(frames[frame-1].At) / time.Second)
}
