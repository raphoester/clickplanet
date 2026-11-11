package parts

import (
	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/allegiances"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/arenas"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/charges"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/seeds"
)

// Grants is the offers' half of bonuses, declared here since bonuses reads no package of planet's: the boxes and
// quizzes, what they may be and how much they grant. The wallet's half is charges.Part.
var Grants = rules.PartOf[bonuses.Rules]{Key: "bonuses", Read: func(m *planetv1.Rules) bonuses.Rules {
	return bonuses.RulesOf(m.GetBonuses())
}}

// All is the one list of the game's parts: a concept that reads the rules adds its Part here.
func All() rules.Parts {
	return rules.Parts{
		clicks.ThrottlePart, clicks.TollPart, clicks.FreezePart, allegiances.LockPart, Grants, charges.Part,
		arenas.Part, seeds.Part, clicks.ReachPart, clicks.CapitalsPart,
	}
}
