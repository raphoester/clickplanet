package ledger

import (
	"sort"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot"
)

const defaultLimit = 20

type Player struct {
	Scope   string
	Account string
	Tiles   int
	Takes   int
	FirstAt time.Time
	LastAt  time.Time

	Banned      bool
	BannedUntil time.Time
	Offence     int
}

func NewTally(counts func(Taking) bool) *Tally {
	return &Tally{counts: counts, callers: make(map[Caller]int), tiles: make(map[uint32]hold)}
}

type Tally struct {
	counts  func(Taking) bool
	callers map[Caller]int
	players []Player
	tiles   map[uint32]hold
}

type hold struct {
	player  int
	country string
}

func (t *Tally) See(taking Taking) {
	if taking.Bombed || !t.counts(taking) {
		delete(t.tiles, taking.Tile)
		return
	}

	key := Caller{Scope: taking.Scope, Account: taking.Account}

	index, ok := t.callers[key]
	if !ok {
		index = len(t.players)
		t.callers[key] = index
		t.players = append(t.players, Player{
			Scope: taking.Scope, Account: taking.Account, FirstAt: taking.At, LastAt: taking.At,
		})
	}

	player := &t.players[index]
	player.Takes++
	if taking.At.Before(player.FirstAt) {
		player.FirstAt = taking.At
	}
	if taking.At.After(player.LastAt) {
		player.LastAt = taking.At
	}

	if taking.Cleared() {
		delete(t.tiles, taking.Tile)
		return
	}

	t.tiles[taking.Tile] = hold{player: index, country: taking.Country}
}

func (t *Tally) Players(owners Owners) []Player {
	for tile, hold := range t.tiles {
		if owner, _ := owners.Owner(tile); owner == hold.country {
			t.players[hold.player].Tiles++
		}
	}

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

func (p *Player) Serving(sentence antibot.Sentence) {
	p.Banned = true
	p.BannedUntil = sentence.Until
	p.Offence = sentence.Offence
}
