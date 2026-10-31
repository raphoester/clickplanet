package finales

import (
	"errors"
	"fmt"
	"slices"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
)

type Season uint32

type Country string

type BookID string

type RoundID string

var (
	ErrNoBook    = errors.New("no book id")
	ErrNoRound   = errors.New("no round id")
	ErrNoCountry = errors.New("no country")
)

func BookIDOf(id string) (BookID, error) {
	if id == "" {
		return "", ErrNoBook
	}
	return BookID(id), nil
}

func (b BookID) String() string { return string(b) }

func RoundIDOf(id string) (RoundID, error) {
	if id == "" {
		return "", ErrNoRound
	}
	return RoundID(id), nil
}

func (r RoundID) String() string { return string(r) }

type Place struct {
	Country Country
	Rank    uint32
}

type Fighter struct {
	Account players.AccountID
	Country Country
}

type Result struct {
	Season   Season
	Table    []Place
	Fighters []Fighter
}

func (r Result) Fights() ([]Fight, error) {
	fights := make([]Fight, 0, len(r.Fighters))
	for _, fighter := range r.Fighters {
		fight, err := FightSnapshot{
			Account: fighter.Account, Season: r.Season, Country: fighter.Country, Rank: r.rankOf(fighter.Country),
		}.Restore()
		if err != nil {
			return nil, err
		}
		fights = append(fights, fight)
	}
	return fights, nil
}

func (r Result) rankOf(country Country) uint32 {
	at := slices.IndexFunc(r.Table, func(place Place) bool { return place.Country == country })
	if at < 0 {
		return 0
	}
	return r.Table[at].Rank
}

type FightSnapshot struct {
	Account players.AccountID
	Season  Season
	Country Country
	// The flag's place on the season's table; 0 when it is not on it.
	Rank uint32
}

func (s FightSnapshot) Restore() (Fight, error) {
	if s.Country == "" {
		return Fight{}, fmt.Errorf("%w: a fight of season %d", ErrNoCountry, s.Season)
	}
	return Fight{snapshot: s}, nil
}

// Fight is an account that took tiles in a season's finale, under the flag it took the most for.
type Fight struct {
	snapshot FightSnapshot
}

func (f Fight) TakeSnapshot() FightSnapshot { return f.snapshot }

func (f Fight) Account() players.AccountID { return f.snapshot.Account }

type Fights []Fight

func (f Fights) FoughtIn(season Season) bool {
	return slices.ContainsFunc(f, func(fight Fight) bool { return fight.snapshot.Season == season })
}

func (f Fights) PlacedIn(season Season, rank uint32) bool {
	return slices.ContainsFunc(f, func(fight Fight) bool {
		return fight.snapshot.Season == season && fight.snapshot.Rank == rank
	})
}
