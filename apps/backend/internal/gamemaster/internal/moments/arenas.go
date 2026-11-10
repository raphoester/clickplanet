package moments

import (
	"cmp"
	"maps"
	"slices"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/schedule"
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

// Judged is the plan that knows these battles judged too; a plan that knows no arena battles stays so.
func (p Plan) Judged(judgements ...Judgement) Plan {
	if p.known.modes == nil || len(judgements) == 0 {
		return p
	}
	modes := maps.Clone(p.known.modes)
	for _, judgement := range judgements {
		modes[judgement.Round] = judgement.Mode
	}
	p.known.modes = modes
	return p
}

// SetWaits is true while planet must not play the set of the battle the turn plays at a time: the battle is not judged
// yet, or it is an arena battle whose arena planet has not told opened, so its set would play on the map from before
// it. A turn of no battle of the plan, and a plan that knows no arena battles, wait for nothing.
func (p Plan) SetWaits(turn schedule.TurnSnapshot, at time.Time) bool {
	battle, ok := p.battleOf(turn, at)
	if !ok || p.known.modes == nil {
		return false
	}
	round := battle.Round()
	mode, judged := p.known.modes[round]
	if !judged {
		return true
	}
	return mode == Arena && !p.known.opened.Contains(round) && !p.known.closed.Contains(round)
}

func (p Plan) battleOf(turn schedule.TurnSnapshot, at time.Time) (Battle, bool) {
	if turn.Kind != schedule.Battle {
		return Battle{}, false
	}
	for _, book := range p.books {
		for _, battle := range book.Battles {
			if battle.runs(at) && battle.EndsSeason == turn.Finale && (turn.Finale || battle.Entry == turn.Entry) {
				return battle, true
			}
		}
	}
	return Battle{}, false
}

// Waits is true while an arena battle that ended at or before a time has its arena open: the map from before it is not back
// yet, so nothing may open on it.
func (p Plan) Waits(at time.Time) bool {
	for _, book := range p.books {
		for _, battle := range book.Battles {
			round := battle.Round()
			if p.known.modes[round] == Arena && !battle.EndsAt.After(at) && !p.known.closed.Contains(round) {
				return true
			}
		}
	}
	return false
}

// Unjudged is the battle that runs at a time and has no mode yet.
func (p Plan) Unjudged(at time.Time) (Battle, bool) {
	battle, ok := p.BattleAt(at)
	if !ok || p.known.modes[battle.Round()] != 0 {
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
// close. The arena is named by the battle's round; an open names the battle's set, which says how its map starts.
type ArenaCommand struct {
	Action ArenaAction
	Arena  RoundID
	At     time.Time
	EndsAt time.Time
	Set    schedule.SetName
}

// ArenaKey is a command's identity: an open is sent once per process, a close again until planet answers it.
type ArenaKey struct {
	Action ArenaAction
	Arena  RoundID
}

func (c ArenaCommand) Key() ArenaKey { return ArenaKey{Action: c.Action, Arena: c.Arena} }

// Closes is true for the command planet answers with a fact: it is sent again until the fact comes.
func (a ArenaAction) Closes() bool { return a == CloseArena }

// ArenasDueAt is every arena command whose time has come. An arena opens while its battle runs, once scoring answered
// the round before it, scored on the map as it is or missed, and no arena battle before it waits for its map, until
// planet tells it opened; it closes once the arena battle ended and scoring answered its round, scored on the arena or
// missed. A close is due until planet tells the arena closed.
func (p Plan) ArenasDueAt(now time.Time) []ArenaCommand {
	var due []ArenaCommand
	for _, book := range p.books {
		for _, battle := range book.Battles {
			round := battle.Round()
			if p.known.modes[round] != Arena || p.known.closed.Contains(round) {
				continue
			}
			switch {
			case battle.runs(now) && !p.known.opened.Contains(round):
				if before, ok := p.roundBefore(battle); ok && p.settled(book.ID(), before.id) && !p.Waits(battle.StartsAt) {
					due = append(due, ArenaCommand{
						Action: OpenArena, Arena: round, At: battle.StartsAt, EndsAt: battle.EndsAt, Set: battle.Set,
					})
				}
			case !now.Before(battle.EndsAt) && p.settled(book.ID(), round):
				due = append(due, ArenaCommand{Action: CloseArena, Arena: round, At: battle.EndsAt, EndsAt: battle.EndsAt})
			}
		}
	}
	slices.SortStableFunc(due, func(a, b ArenaCommand) int {
		return cmp.Or(a.At.Compare(b.At), cmp.Compare(a.Action, b.Action))
	})
	return due
}
