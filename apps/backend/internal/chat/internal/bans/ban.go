// Package bans is the chat's own moderation: who a person silenced, and what
// that does to what they already said.
//
// It is not the antibot's shadow ban. That one is passed by a watchdog, keys on
// a click scope and drops clicks; this one is passed by a person, keys on the
// author tag the chat shows, and reaches the chat and nothing else.
package bans

import (
	"errors"
	"time"
)

var ErrNotBanned = errors.New("this member is not banned")

// Ban is one member a person silenced. It has no expiry: a call a person made is one a person lifts.
type Ban struct {
	AuthorTag string
	BannedAt  time.Time
	Reason    string
}
