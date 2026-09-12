package cptime

import "time"

// Clock is the only way the rest of the backend asks what time it is. Nothing
// below calls time.Now directly, so every deadline, window and expiry can be
// driven from a test.
type Clock interface {
	Now() time.Time
}

// SystemClock is the real one: the wall clock the process runs on.
type SystemClock struct{}

func (c SystemClock) Now() time.Time {
	return time.Now()
}
