package subscribers

import "time"

// An account's takes are never cut, so the update that anonymizes them grows with its history.
const Timeout = time.Minute
