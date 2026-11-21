package territories_query

import (
	"cmp"
	"slices"

	"google.golang.org/protobuf/types/known/timestamppb"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

type Tally interface {
	Tally() (clicks.MapVersion, clicks.Tally)
}

type ShieldTally interface {
	ShieldTally() []clicks.FlagShields
}

type OutFlags interface {
	OutFlags() []clicks.OutFlag
}

// Board is the board in play: its name, its tile count, and whose ground each landmass is.
type Board interface {
	Name() clicks.BoardName
	Tiles() uint32
	CountryOfLandmass(landmass clicks.LandmassID) string
}

func NewMemoryQuery(tally Tally, shields ShieldTally, out OutFlags, board Board) *MemoryQuery {
	return &MemoryQuery{tally: tally, shields: shields, out: out, board: board}
}

type MemoryQuery struct {
	tally   Tally
	shields ShieldTally
	out     OutFlags
	board   Board
}

func (q *MemoryQuery) Territories() *planetv1.GetTerritoriesResponse {
	_, tally := q.tally.Tally()

	// A tile on no country's land counts for its flag and lies on no ground.
	grounds := make(map[string]map[string]uint32)
	for _, row := range tally.Landmasses {
		ground := q.board.CountryOfLandmass(row.Landmass)
		if ground == "" {
			continue
		}
		for _, held := range row.Flags {
			if grounds[held.Flag] == nil {
				grounds[held.Flag] = make(map[string]uint32)
			}
			grounds[held.Flag][ground] += held.Tiles
		}
	}

	shields := make(map[string]uint32)
	for _, count := range q.shields.ShieldTally() {
		shields[count.Flag] = count.Shields
	}

	territories := make([]*planetv1.Territory, 0, len(tally.Countries))
	for _, count := range tally.Countries {
		territories = append(territories, &planetv1.Territory{
			CountryId: count.Flag, Tiles: count.Tiles, Grounds: groundsOf(grounds[count.Flag]), Shields: shields[count.Flag],
		})
	}
	slices.SortFunc(territories, func(a, b *planetv1.Territory) int {
		return cmp.Compare(a.GetCountryId(), b.GetCountryId())
	})

	out := q.out.OutFlags()
	knockedOut := make([]*planetv1.KnockedOutFlag, 0, len(out))
	for _, flag := range out {
		knockedOut = append(knockedOut, &planetv1.KnockedOutFlag{CountryId: flag.Flag, KnockedOutAt: timestamppb.New(flag.At)})
	}

	return &planetv1.GetTerritoriesResponse{
		Tiles: q.board.Tiles(), Territories: territories, Board: string(q.board.Name()), KnockedOut: knockedOut,
	}
}

func groundsOf(held map[string]uint32) []*planetv1.Ground {
	grounds := make([]*planetv1.Ground, 0, len(held))
	for country, tiles := range held {
		grounds = append(grounds, &planetv1.Ground{CountryId: country, Tiles: tiles})
	}
	slices.SortFunc(grounds, func(a, b *planetv1.Ground) int {
		return cmp.Compare(a.GetCountryId(), b.GetCountryId())
	})
	return grounds
}
