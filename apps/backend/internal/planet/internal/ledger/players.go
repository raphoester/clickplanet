package ledger

import (
	"sort"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

const defaultLimit = 20

type Player struct {
	Scope   string
	Account string
	Flag    string
	Tiles   int
	Takes   int
	FirstAt time.Time
	LastAt  time.Time

	RefusedUntil time.Time
}

func NewTally(counts func(Taking) bool) *Tally {
	return &Tally{
		counts: counts, callers: make(map[Caller]int), tiles: make(map[uint32]hold), flags: make(map[hold]int),
	}
}

type Tally struct {
	counts  func(Taking) bool
	callers map[Caller]int
	players []Player
	tiles   map[uint32]hold
	flags   map[hold]int
}

type hold struct {
	player  int
	country string
}

func (t *Tally) See(event Event) {
	player := -1
	event.Replay(func(taking Taking) {
		if !t.counts(taking) {
			delete(t.tiles, taking.Tile)
			return
		}

		key := Caller{Scope: taking.Scope}
		if taking.Account != "" {
			key = Caller{Account: taking.Account}
		}
		index, ok := t.callers[key]
		if !ok {
			index = len(t.players)
			t.callers[key] = index
			t.players = append(t.players, Player{
				Scope: taking.Scope, Account: taking.Account, FirstAt: taking.At, LastAt: taking.At,
			})
		}
		player = index

		if taking.At.Before(t.players[index].FirstAt) {
			t.players[index].FirstAt = taking.At
		}
		if !taking.At.Before(t.players[index].LastAt) {
			t.players[index].LastAt = taking.At
			t.players[index].Scope = taking.Scope
		}

		if taking.Cleared() {
			delete(t.tiles, taking.Tile)
			return
		}
		paint := hold{player: index, country: taking.Country}
		t.tiles[taking.Tile] = paint
		t.flags[paint]++
	})

	if player >= 0 {
		t.players[player].Takes++
	}
}

func (t *Tally) Players(owners Owners) []Player {
	for tile, hold := range t.tiles {
		if owner, _ := owners.Owner(tile); owner == hold.country {
			t.players[hold.player].Tiles++
		}
	}
	t.pickFlags()

	players := t.players
	sort.Slice(players, func(i, j int) bool {
		if !players[i].LastAt.Equal(players[j].LastAt) {
			return players[i].LastAt.After(players[j].LastAt)
		}
		if players[i].Scope != players[j].Scope {
			return players[i].Scope < players[j].Scope
		}
		return players[i].Account < players[j].Account
	})

	return players
}

// A player's flag is the one it took the most tiles for, and the first by id of those it took as many for.
func (t *Tally) pickFlags() {
	most := make([]int, len(t.players))
	for paint, tiles := range t.flags {
		player := &t.players[paint.player]
		if tiles > most[paint.player] || tiles == most[paint.player] && paint.country < player.Flag {
			most[paint.player], player.Flag = tiles, paint.country
		}
	}
}

func ByTakes(players []Player) []Player {
	sort.SliceStable(players, func(i, j int) bool {
		if players[i].Takes != players[j].Takes {
			return players[i].Takes > players[j].Takes
		}
		return players[i].Tiles > players[j].Tiles
	})

	return players
}

func Top(players []Player, limit int) []Player {
	if limit <= 0 {
		limit = defaultLimit
	}

	return players[:min(limit, len(players))]
}

func (p Player) ActiveFor() time.Duration {
	return p.LastAt.Sub(p.FirstAt)
}

func (p Player) TilesPerMinute() float64 {
	return p.perMinute(p.Tiles)
}

func (p Player) TakesPerMinute() float64 {
	return p.perMinute(p.Takes)
}

func (p Player) perMinute(count int) float64 {
	active := p.ActiveFor()
	if active <= 0 {
		return 0
	}

	return float64(count) / active.Minutes()
}

func (p *Player) Refused(refusal clicks.Refusal) {
	p.RefusedUntil = refusal.Until()
}
