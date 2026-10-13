package subscribers

import "time"

// Not 5s as elsewhere: an account's notes may carry files, and each is one call to the bucket.
const Timeout = 30 * time.Second
