package inprocess_rename_rule

import (
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/playermessage"
)

func New() Rule {
	return Rule{}
}

type Rule struct{}

func (Rule) RenamableAt(chosenAt time.Time) int64 {
	return playermessage.RenamableAt(chosenAt)
}
