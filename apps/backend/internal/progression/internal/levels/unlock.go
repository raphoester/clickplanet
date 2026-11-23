package levels

// Kind is a kind of charge a level unlocks, in planet's words: planet.v1.BonusKind without its prefix, in lower case.
type Kind string

// Unlock is one rung of the ladder: from its level on, a box or a right answer may be drawn as its kind while the
// bonuses' gates are on. One type per rung, as the title tracks are.
type Unlock interface {
	Kind() Kind
	Level() Level
}

// SpreadUnlock comes with the first tile a linked account takes: the first take of a day alone is 100 XP.
type SpreadUnlock struct{}

func (SpreadUnlock) Kind() Kind   { return "spread_clicks" }
func (SpreadUnlock) Level() Level { return 2 }

// ShieldsUnlock comes in the first minute, about a dozen tiles after the first.
type ShieldsUnlock struct{}

func (ShieldsUnlock) Kind() Kind   { return "shields" }
func (ShieldsUnlock) Level() Level { return 3 }

// EncloseUnlock comes with a good first day.
type EncloseUnlock struct{}

func (EncloseUnlock) Kind() Kind   { return "enclose_clicks" }
func (EncloseUnlock) Level() Level { return 5 }

// BombUnlock comes about day 3 for a regular player.
type BombUnlock struct{}

func (BombUnlock) Kind() Kind   { return "bomb" }
func (BombUnlock) Level() Level { return 7 }

// VirusUnlock comes after about a week for a regular player, once the bomb and the enclosure are known: the first
// charge that plays on its own.
type VirusUnlock struct{}

func (VirusUnlock) Kind() Kind   { return "virus" }
func (VirusUnlock) Level() Level { return 10 }

// MineUnlock comes about day 19 for a regular player: the first charge that waits for the enemy.
type MineUnlock struct{}

func (MineUnlock) Kind() Kind   { return "mine" }
func (MineUnlock) Level() Level { return 13 }

// RallyUnlock comes after about a month for a regular player: the first charge a flag plays together.
type RallyUnlock struct{}

func (RallyUnlock) Kind() Kind   { return "rally" }
func (RallyUnlock) Level() Level { return 15 }

// TrapUnlock comes last, after about two months and a half for a regular player (96,000 XP): the first charge that
// costs the enemy clicks, not land.
type TrapUnlock struct{}

func (TrapUnlock) Kind() Kind   { return "trap" }
func (TrapUnlock) Level() Level { return 20 }

// Ladder is every kind a level unlocks. A kind not on it unlocks at no level: whether anyone may draw it is the bonuses'
// open kinds.
type Ladder struct{}

func (Ladder) Unlocks() []Unlock {
	return []Unlock{
		SpreadUnlock{},
		ShieldsUnlock{},
		EncloseUnlock{},
		BombUnlock{},
		VirusUnlock{},
		MineUnlock{},
		RallyUnlock{},
		TrapUnlock{},
	}
}

// UnlockedAt is every kind unlocked at the level, in the ladder's order: none for no level.
func (l Ladder) UnlockedAt(level Level) []Kind {
	kinds := []Kind{}
	for _, unlock := range l.Unlocks() {
		if unlock.Level() <= level {
			kinds = append(kinds, unlock.Kind())
		}
	}
	return kinds
}
