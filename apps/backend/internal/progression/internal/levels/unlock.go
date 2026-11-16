package levels

// Kind is a kind of charge a level unlocks, in planet's words: planet.v1.BonusKind without its prefix, in lower case.
type Kind string

// Unlock is one rung of the ladder: from its level on, a box or a right answer may be drawn as its kind while the
// bonuses' gates are on. One type per rung, as the title tracks are.
type Unlock interface {
	Kind() Kind
	Level() Level
}

// EncloseUnlock comes about 5 minutes in.
type EncloseUnlock struct{}

func (EncloseUnlock) Kind() Kind   { return "enclose_clicks" }
func (EncloseUnlock) Level() Level { return 4 }

// BombUnlock comes with a good first day.
type BombUnlock struct{}

func (BombUnlock) Kind() Kind   { return "bomb" }
func (BombUnlock) Level() Level { return 5 }

// VirusUnlock comes once the bomb and the enclosure are known: the first charge that plays on its own.
type VirusUnlock struct{}

func (VirusUnlock) Kind() Kind   { return "virus" }
func (VirusUnlock) Level() Level { return 6 }

// MineUnlock comes after the virus, about day 4 for a regular player: the first charge that waits for the enemy.
type MineUnlock struct{}

func (MineUnlock) Kind() Kind   { return "mine" }
func (MineUnlock) Level() Level { return 8 }

// Ladder is every kind a level unlocks. A kind not on it unlocks at no level: whether anyone may draw it is the bonuses'
// open kinds.
type Ladder struct{}

func (Ladder) Unlocks() []Unlock {
	return []Unlock{EncloseUnlock{}, BombUnlock{}, VirusUnlock{}, MineUnlock{}}
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
