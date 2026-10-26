package parts

import (
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules"
)

// All is the one list of the game's parts: a concept that reads the rules adds its Part here.
func All() rules.Parts {
	return rules.Parts{clicks.ThrottlePart, clicks.TollPart, bonuses.Part}
}
