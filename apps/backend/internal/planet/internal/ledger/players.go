package ledger

import (
	"sort"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot"
)

const defaultLimit = 20

type Player struct {
	Scope string
	// Tiles is how many of the takes gathered into this player the scope made.
	Tiles   int
	FirstAt time.Time
	LastAt  time.Time

	Banned      bool
	BannedUntil time.Time
	Offence     int
}

// Players gathers takes by scope, latest take first, and scope order between equal times.
func Players(takings []Taking) []Player {
	byScope := make(map[string]*Player)
	for _, taking := range takings {
		player, ok := byScope[taking.Scope]
		if !ok {
			player = &Player{Scope: taking.Scope, FirstAt: taking.At, LastAt: taking.At}
			byScope[taking.Scope] = player
		}

		player.Tiles++
		if taking.At.Before(player.FirstAt) {
			player.FirstAt = taking.At
		}
		if taking.At.After(player.LastAt) {
			player.LastAt = taking.At
		}
	}

	players := make([]Player, 0, len(byScope))
	for _, player := range byScope {
		players = append(players, *player)
	}

	sort.Slice(players, func(i, j int) bool {
		if !players[i].LastAt.Equal(players[j].LastAt) {
			return players[i].LastAt.After(players[j].LastAt)
		}
		return players[i].Scope < players[j].Scope
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

// Serving marks the player as under a running ban.
func (p *Player) Serving(sentence antibot.Sentence) {
	p.Banned = true
	p.BannedUntil = sentence.Until
	p.Offence = sentence.Offence
}
