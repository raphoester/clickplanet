package subscribers

import "time"

// A payment is a call or two to planet, which answers at once.
const Timeout = 30 * time.Second
