package parts

import (
	boxesv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/boxes/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/boxes/internal/boxes"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cprules"
)

// All is the one list of the boxes' parts: a concept that reads the rules adds its Part here.
func All() cprules.Parts[*boxesv1.Rules] {
	return cprules.Parts[*boxesv1.Rules]{boxes.Part}
}
