//go:build testing

package ledger

import "github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"

// Staged is an event made on a stage, for a test that writes the ledger's rows.
type Staged struct {
	Event
	On clicks.Stage
}

// StageOf is the stage an event of a test was made on: the world's map on TestBoard unless Staged says another.
func StageOf(event Event) clicks.Stage {
	if staged, ok := event.(Staged); ok {
		return staged.On
	}
	return clicks.WorldStage
}
