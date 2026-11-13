package subscribers

import "time"

// A forgotten account is one update of the answers it gave.
const Timeout = 30 * time.Second
