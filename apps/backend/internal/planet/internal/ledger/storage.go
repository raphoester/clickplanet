package ledger

import (
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

// Storage keeps every event with its stage: the map it was made on, on the board that map played.
type Storage interface {
	Append(on clicks.Stage, event Event)
	Replay(see func(on clicks.Stage, event Event)) Position
	Forget(caller Caller, before Position)
	ForgetBefore(cutoff time.Time)
}

// Stages hands a write the stage it lands on: no swap of the map or of its board comes between the write and what is
// recorded of it.
type Stages interface {
	During(write func(on clicks.Stage))
}
