package ledger

import "time"

type Storage interface {
	Append(taking Taking)
	// Replay hands see every take not forgotten, oldest first, and returns the position after the last.
	Replay(see func(Taking)) Position
	Forget(scope string, before Position)
	ForgetBefore(cutoff time.Time)
}
