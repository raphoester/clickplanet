package ledger

import "time"

type Storage interface {
	Append(taking Taking)
	AppendBombing(bombing Bombing)
	Replay(see func(Taking)) Position
	Forget(caller Caller, before Position)
	ForgetBefore(cutoff time.Time)
}
