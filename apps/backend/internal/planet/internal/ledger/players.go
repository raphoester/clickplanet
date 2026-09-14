package ledger

import (
	"sort"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot"
)

const defaultLimit = 20

type Player struct {
	Scope string
	// Tiles is what the scope still holds; Takes counts every take, so a painted-over bot still shows.
	Tiles   int
	Takes   int
	FirstAt time.Time
	LastAt  time.Time

	Banned      bool
	BannedUntil time.Time
	Offence     int
}

func NewTally(counts func(Taking) bool) *Tally {
	return &Tally{counts: counts, scopes: make(map[string]int), tiles: make(map[uint32]hold)}
}

type Tally struct {
	counts  func(Taking) bool
	scopes  map[string]int
	players []Player
	tiles   map[uint32]hold
}

type hold struct {
	player  int
	country string
}

func (t *Tally) See(taking Taking) {
	if !t.counts(taking) {
		delete(t.tiles, taking.Tile)
		return
	}

	index, ok := t.scopes[taking.Scope]
	if !ok {
		index = len(t.players)
		t.scopes[taking.Scope] = index
		t.players = append(t.players, Player{Scope: taking.Scope, FirstAt: taking.At, LastAt: taking.At})
	}

	player := &t.players[index]
	player.Takes++
	if taking.At.Before(player.FirstAt) {
		player.FirstAt = taking.At
	}
	if taking.At.After(player.LastAt) {
		player.LastAt = taking.At
	}

	t.tiles[taking.Tile] = hold{player: index, country: taking.Country}
}

// Players is every scope with a take that counts, latest take first, then scope order.
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
		return players[i].Scope < players[j].Scope
	})

	return players
}

// ByTakes orders players most takes first, then most tiles held, keeping the order between equals.
func ByTakes(players []Player) []Player {
	sort.SliceStable(players, func(i, j int) bool {
		if players[i].Takes != players[j].Takes {
			return players[i].Takes > players[j].Takes
		}
		return players[i].Tiles > players[j].Tiles
	})

	return players
}

// Top cuts players to limit; zero or less is the default.
func Top(players []Player, limit int) []Player {
	if limit <= 0 {
		limit = defaultLimit
	}

	return players[:min(limit, len(players))]
}

// ActiveFor is the time from the player's first take to its last.
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

// Serving marks the player as under a running ban.
func (p *Player) Serving(sentence antibot.Sentence) {
	p.Banned = true
	p.BannedUntil = sentence.Until
	p.Offence = sentence.Offence
}
