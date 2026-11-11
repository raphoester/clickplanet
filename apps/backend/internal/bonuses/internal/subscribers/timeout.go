package subscribers

import "time"

// A payment is one call to planet, which answers at once.
const Timeout = 30 * time.Second
