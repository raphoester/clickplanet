// Package subscribers is the chat module's edge for events, as chatv1controller is its edge for the wire.
package subscribers

import "time"

// Timeout bounds one event's write, so a stuck database cannot hold a subscriber and fill its buffer forever.
const Timeout = 5 * time.Second
