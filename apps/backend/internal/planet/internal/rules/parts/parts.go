package parts

import (
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/allegiances"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/arenas"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/charges"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/seeds"
)

// All is the one list of the game's parts: a concept that reads the rules adds its Part here.
func All() rules.Parts {
	return rules.Parts{
		clicks.ThrottlePart, clicks.TollPart, clicks.FreezePart, allegiances.LockPart, charges.Part,
		arenas.Part, seeds.Part, clicks.ReachPart, clicks.CapitalsPart,
	}
}
