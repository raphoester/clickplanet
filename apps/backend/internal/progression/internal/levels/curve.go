package levels

import (
	"math"
	"math/bits"
)

// Level has no top. An account with no XP is at level 1; level 0 is no level at all, a guest's.
type Level uint32

// Curve is cubic: the XP to reach level L is 14 × (L − 1)³, so going from L to L + 1 costs 14 × (3L² − 3L + 1).
// Levels come fast in the first days and slower after that.
type Curve struct{}

const factor = 14

// Threshold is the XP to reach the level. A level no XP can reach costs the most an XP holds.
func (c Curve) Threshold(level Level) XP {
	threshold, reachable := c.threshold(level)
	if !reachable {
		return math.MaxUint64
	}
	return threshold
}

// LevelOf is the highest level whose threshold the XP reaches, at least 1.
func (c Curve) LevelOf(xp XP) Level {
	level := Level(math.Cbrt(float64(xp)/factor)) + 1
	for level > 1 && !c.reached(level, xp) {
		level--
	}
	for c.reached(level+1, xp) {
		level++
	}
	return level
}

func (c Curve) StandingOf(xp XP) Standing {
	return c.StandingAt(c.LevelOf(xp), xp)
}

func (c Curve) StandingAt(level Level, xp XP) Standing {
	return Standing{level: level, xp: xp, levelXP: c.Threshold(level), nextLevelXP: c.Threshold(level + 1)}
}

func (c Curve) reached(level Level, xp XP) bool {
	threshold, reachable := c.threshold(level)
	return reachable && threshold <= xp
}

func (Curve) threshold(level Level) (XP, bool) {
	if level <= 1 {
		return 0, true
	}
	steps := uint64(level - 1)
	product := uint64(factor)
	for range 3 {
		high, low := bits.Mul64(product, steps)
		if high != 0 {
			return 0, false
		}
		product = low
	}
	return XP(product), true
}

// Standing is where an account stands on the curve: its level, its XP, and the two thresholds its bar runs between.
type Standing struct {
	level       Level
	xp          XP
	levelXP     XP
	nextLevelXP XP
}

func (s Standing) Level() Level { return s.level }

func (s Standing) XP() XP { return s.xp }

func (s Standing) LevelXP() XP { return s.levelXP }

func (s Standing) NextLevelXP() XP { return s.nextLevelXP }

// A guest's standing is empty: no XP and no level.
func (s Standing) Empty() bool { return s.level == 0 }
