package parts

import (
	bonusesv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/bonuses/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cprules"
)

// All is the one list of the offers' parts: a concept that reads the rules adds its Part here.
func All() cprules.Parts[*bonusesv1.Rules] {
	return cprules.Parts[*bonusesv1.Rules]{bonuses.Part}
}
