package subscribers

import "time"

// A payment is one call to planet, which answers at once; a forgotten account is one update of the answers it gave.
const Timeout = 30 * time.Second
