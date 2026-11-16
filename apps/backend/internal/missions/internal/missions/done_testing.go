//go:build testing

package missions

// Every kind, one per Order type.
func TestingKinds() []Kind {
	return []Kind{
		KindTakeFor, KindTakeFrom, KindTakeIn, KindTakeFromIn, KindCatchBoxes, KindAnswerQuizzes, KindBombIn,
		KindShieldTiles, KindFortifyTiles,
	}
}
