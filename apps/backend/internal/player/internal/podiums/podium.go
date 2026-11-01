package podiums

import (
	"errors"
	"fmt"
	"slices"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/finales"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
)

const lowestStep = 3

var ErrOffThePodium = errors.New("a rank off the podium")

type Entry struct {
	Account players.AccountID
	Rank    uint32
}

// Board is a season's players, ranked as scoring ranks its board: ties share a rank.
type Board struct {
	Season  finales.Season
	Entries []Entry
}

func (b Board) Places() []Place {
	var places []Place
	for _, entry := range b.Entries {
		if onThePodium(entry.Rank) {
			places = append(places, Place{snapshot: PlaceSnapshot{Account: entry.Account, Season: b.Season, Rank: entry.Rank}})
		}
	}
	return places
}

func onThePodium(rank uint32) bool { return rank >= 1 && rank <= lowestStep }

type PlaceSnapshot struct {
	Account players.AccountID
	Season  finales.Season
	Rank    uint32
}

func (s PlaceSnapshot) Restore() (Place, error) {
	if !onThePodium(s.Rank) {
		return Place{}, fmt.Errorf("%w: %d in season %d", ErrOffThePodium, s.Rank, s.Season)
	}
	return Place{snapshot: s}, nil
}

// Place is an account ranked 1st to 3rd on a season's board.
type Place struct {
	snapshot PlaceSnapshot
}

func (p Place) TakeSnapshot() PlaceSnapshot { return p.snapshot }

func (p Place) Account() players.AccountID { return p.snapshot.Account }

type Places []Place

func (p Places) RankedIn(season finales.Season, rank uint32) bool {
	return slices.ContainsFunc(p, func(place Place) bool {
		return place.snapshot.Season == season && place.snapshot.Rank == rank
	})
}
