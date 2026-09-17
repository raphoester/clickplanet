package ledger

import "time"

type Storage interface {
	Append(taking Taking)
	// Replay hands see every take not forgotten, oldest first, and returns the position after the last.
	Replay(see func(Taking)) Position
	// Forget hides the caller's takes before the position from every replay after.
	Forget(caller Caller, before Position)
	ForgetBefore(cutoff time.Time)
}
