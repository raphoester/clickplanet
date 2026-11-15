package levels

import "github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/accounts"

// Scale weighs what an account did into XP and places it on the curve.
type Scale struct{}

func (Scale) Sources() []Source {
	return []Source{
		TilesTaken{},
		DaysPlayed{},
		MissionsDone{},
		BombsDropped{},
		BoxesCaught{},
		RightAnswers{},
		TilesFortified{},
		MessagesSent{},
	}
}

// StandingOf is where the account's record puts it. A guest has no XP and no level, whatever it did.
func (s Scale) StandingOf(account accounts.Account, record Record) Standing {
	if !account.Linked() {
		return Standing{}
	}
	var xp XP
	for _, source := range s.Sources() {
		xp += source.XP(record)
	}
	return Curve{}.StandingOf(xp)
}
