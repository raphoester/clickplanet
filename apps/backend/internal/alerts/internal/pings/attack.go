package pings

import (
	"sync"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/nations"
)

// One flag took this many of another's tiles within the window: replayed on the last week of season 0, about ten
// pings a day across thirty flags, the busiest flag once a day. At 50 tiles it was 75 a day.
const (
	AttackTiles  = 200
	AttackWindow = 5 * time.Minute
	// A ping about an older attack is news nobody can act on: a boot that reads a backlog stays quiet.
	Fresh = 10 * time.Minute
)

type Take struct {
	Victim   nations.Flag
	Attacker nations.Flag
	Ground   nations.Flag
	At       time.Time
}

type Attack struct {
	Victim   nations.Flag
	Attacker nations.Flag
	// Where most of the tiles were.
	Ground nations.Flag
	Tiles  int
	At     time.Time
}

func (a Attack) Stale(now time.Time) bool {
	return now.Sub(a.At) > Fresh
}

// Kept in memory: a restart forgets the takes of the last minutes, and with them at most one attack.
type Watch struct {
	mu    sync.Mutex
	takes map[nations.Flag][]Take
}

func NewWatch() *Watch {
	return &Watch{takes: map[nations.Flag][]Take{}}
}

func (w *Watch) Hear(take Take) (Attack, bool) {
	w.mu.Lock()
	defer w.mu.Unlock()

	recent := w.takes[take.Victim][:0:0]
	for _, kept := range w.takes[take.Victim] {
		if take.At.Sub(kept.At) < AttackWindow {
			recent = append(recent, kept)
		}
	}
	recent = append(recent, take)

	grounds := map[nations.Flag]int{}
	for _, kept := range recent {
		if kept.Attacker == take.Attacker {
			grounds[kept.Ground]++
		}
	}
	tiles := 0
	for _, count := range grounds {
		tiles += count
	}
	if tiles < AttackTiles {
		w.takes[take.Victim] = recent
		return Attack{}, false
	}

	others := recent[:0:0]
	for _, kept := range recent {
		if kept.Attacker != take.Attacker {
			others = append(others, kept)
		}
	}
	w.takes[take.Victim] = others
	return Attack{Victim: take.Victim, Attacker: take.Attacker, Ground: mostOf(grounds), Tiles: tiles, At: take.At}, true
}

func mostOf(grounds map[nations.Flag]int) nations.Flag {
	var most nations.Flag
	for ground, count := range grounds {
		if count > grounds[most] || (count == grounds[most] && ground < most) {
			most = ground
		}
	}
	return most
}

// A flag is pinged at most once in QuietFor and MostADay times a day: a server that pings all day is muted.
const (
	QuietFor = 3 * time.Hour
	MostADay = 2
	Day      = 24 * time.Hour
)

// A flag's pings since a day.
type History []time.Time

func (h History) Allows(at time.Time) bool {
	today := 0
	for _, sent := range h {
		if at.Sub(sent) < QuietFor {
			return false
		}
		if at.Sub(sent) < Day {
			today++
		}
	}
	return today < MostADay
}
