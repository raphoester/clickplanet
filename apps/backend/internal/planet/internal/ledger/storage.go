package ledger

import "time"

type Storage interface {
	Append(event Event)
	Replay(see func(Event)) Position
	Forget(caller Caller, before Position)
	ForgetBefore(cutoff time.Time)
}
