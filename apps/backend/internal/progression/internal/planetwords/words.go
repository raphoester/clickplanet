// Package planetwords is what progression says in planet's language: the kinds of charge a level unlocks, as
// planet.v1.BonusKind names them.
package planetwords

import (
	"errors"
	"fmt"
	"strings"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/levels"
)

var ErrUnknownKind = errors.New("a kind of charge planet does not name")

func BonusKind(kind levels.Kind) (planetv1.BonusKind, error) {
	theirs, known := planetv1.BonusKind_value["BONUS_KIND_"+strings.ToUpper(string(kind))]
	if !known || theirs == int32(planetv1.BonusKind_BONUS_KIND_UNSPECIFIED) {
		return planetv1.BonusKind_BONUS_KIND_UNSPECIFIED, fmt.Errorf("%w: %q", ErrUnknownKind, kind)
	}
	return planetv1.BonusKind(theirs), nil
}
