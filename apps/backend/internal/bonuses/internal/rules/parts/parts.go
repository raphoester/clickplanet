package parts

import (
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/rules"
)

// All is the one list of the offers' parts: a concept that reads the rules adds its Part here.
func All() rules.Parts {
	return rules.Parts{bonuses.Part}
}
