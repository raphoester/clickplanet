package domain

import "errors"

// ErrAttestationFailed covers every reason a caller was not minted a session.
// Which reason it was is logged, never returned: a caller learns that it was
// refused, not which check it should work on next.
var ErrAttestationFailed = errors.New("attestation failed")
