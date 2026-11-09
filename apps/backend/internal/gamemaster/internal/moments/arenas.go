package moments

import (
	"cmp"
	"slices"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
)

// Mode is how a battle plays, judged once as it starts.
type Mode uint8

const (
	Plain Mode = iota + 1
	// Arena plays on a blank map, which planet calls an arena, and brings the map from before back at its end.
	Arena
)

func (m Mode) String() string {
	switch m {
	case Plain:
		return "plain"
	case Arena:
		return "arena"
	}
	return "unknown"
}

// Judgement is a battle's mode, judged once from its rule set as it starts, and kept: a set edited later changes it no
// more.
type Judgement struct {
	Round RoundID
	Mode  Mode
}

// Arenas is what the gamemaster kept of the arena battles: each battle judged, each round scored, and each
// arena planet closed. The zero Arenas knows none: a plan with it opens every round on time and waits for nothing, as a
// read of the plan needs.
type Arenas struct {
	modes  map[RoundID]Mode
	scored *cpcolls.Set[RoundID]
	closed *cpcolls.Set[RoundID]
}

func NewArenas(judged []Judgement, scored, closed []RoundID) Arenas {
	known := Arenas{modes: map[RoundID]Mode{}, scored: cpcolls.NewSet(scored...), closed: cpcolls.NewSet(closed...)}
	for _, judgement := range judged {
		known.modes[judgement.Round] = judgement.Mode
	}
	return known
}

// With is the plan that knows the arena battles: a battle's round opens once it is judged, and what comes after an arena battle
// waits for its arena to close.
func (p Plan) With(arenas Arenas) Plan {
	p.arenas = arenas
	return p
}

// Waits is true while an arena battle that ended at or before a time has its arena open: the map from before it is not back
// yet, so nothing may open on it.
func (p Plan) Waits(at time.Time) bool {
	for _, book := range p.books {
		for _, battle := range book.Battles {
			round := battle.Round()
			if p.arenas.modes[round] == Arena && !battle.EndsAt.After(at) && !p.arenas.closed.Contains(round) {
				return true
			}
		}
	}
	return false
}

// Unjudged is the battle that runs at a time and has no mode yet.
func (p Plan) Unjudged(at time.Time) (Battle, bool) {
	battle, ok := p.BattleAt(at)
	if !ok || p.arenas.modes[battle.Round()] != 0 {
		return Battle{}, false
	}
	return battle, true
}

// Judge is a battle's mode: an arena battle when its set says so, unless it opens its book, on the map the season's reset
// blanked already.
func (p Plan) Judge(battle Battle, arena bool) Judgement {
	judgement := Judgement{Round: battle.Round(), Mode: Plain}
	if _, ok := p.roundBefore(battle); arena && ok {
		judgement.Mode = Arena
	}
	return judgement
}

// The round of the battle's book that ends as it starts: the day it cut, or a battle just before it.
func (p Plan) roundBefore(battle Battle) (round, bool) {
	for _, book := range p.books {
		if book.Window.Number != battle.Season {
			continue
		}
		for _, r := range book.rounds() {
			if r.endsAt.Equal(battle.StartsAt) && r.id != battle.Round() {
				return r, true
			}
		}
	}
	return round{}, false
}

// ArenaAction is what the gamemaster asks planet for an arena battle; at one instant a close is sent before an open.
type ArenaAction uint8

const (
	CloseArena ArenaAction = iota + 1
	OpenArena
)

func (a ArenaAction) String() string {
	switch a {
	case CloseArena:
		return "close arena"
	case OpenArena:
		return "open arena"
	}
	return "unknown"
}

// ArenaCommand is what the gamemaster asks planet for an arena battle: At is the battle's start for an open, its end for a
// close. The arena is named by the battle's round.
type ArenaCommand struct {
	Action ArenaAction
	Arena  RoundID
	At     time.Time
	EndsAt time.Time
}

// ArenaKey is a command's identity: each is sent once per process.
type ArenaKey struct {
	Action ArenaAction
	Arena  RoundID
}

func (c ArenaCommand) Key() ArenaKey { return ArenaKey{Action: c.Action, Arena: c.Arena} }

// ArenasDueAt is every arena command whose time has come. An arena opens while its battle runs, once the round before
// it is scored on the map as it is and no arena battle before it waits for its map; it closes once the arena battle ended and its
// round is scored on the arena. A close is sent until planet tells the arena closed.
func (p Plan) ArenasDueAt(now time.Time) []ArenaCommand {
	var due []ArenaCommand
	for _, book := range p.books {
		for _, battle := range book.Battles {
			round := battle.Round()
			if p.arenas.modes[round] != Arena || p.arenas.closed.Contains(round) {
				continue
			}
			switch {
			case battle.runs(now):
				if before, ok := p.roundBefore(battle); ok && p.arenas.scored.Contains(before.id) && !p.Waits(battle.StartsAt) {
					due = append(due, ArenaCommand{Action: OpenArena, Arena: round, At: battle.StartsAt, EndsAt: battle.EndsAt})
				}
			case !now.Before(battle.EndsAt) && p.arenas.scored.Contains(round):
				due = append(due, ArenaCommand{Action: CloseArena, Arena: round, At: battle.EndsAt, EndsAt: battle.EndsAt})
			}
		}
	}
	slices.SortStableFunc(due, func(a, b ArenaCommand) int {
		return cmp.Or(a.At.Compare(b.At), cmp.Compare(a.Action, b.Action))
	})
	return due
}
