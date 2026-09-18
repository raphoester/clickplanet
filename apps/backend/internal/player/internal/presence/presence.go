// Package presence is who is playing: the players who announced themselves lately, and under which flag.
//
// It is kept in memory and nowhere else. A restart empties it, and every client fills it again within one
// announce interval.
package presence

import (
	"cmp"
	"errors"
	"slices"
	"strings"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
)

// TTL is how long a visit counts after its announce. A client announces every 30s, and a hidden tab's timers
// may fire once a minute, so a live player is never dropped between two announces.
const TTL = 90 * time.Second

var ErrUnknownCountry = errors.New("unknown country")

// Key names a line of the roster for as long as its player stays on it, a sign-in and a new name included.
// It is handed to anybody, so it says nothing about the account.
type Key string

// Visit is a player's last announce.
type Visit struct {
	Account players.AccountID
	// Key is the storage's, kept from the account's first announce until the visit is gone.
	Key Key
	// Author is the name the roster shows: the username, or the guest code.
	Author players.Author
	// Tag is the address the visit was announced from. It never leaves the server: the storage caps the visits
	// of one address with it.
	Tag     players.Tag
	Country string
	At      time.Time
}

// Fresh is whether the visit still counts at now.
func (v Visit) Fresh(now time.Time) bool {
	return now.Sub(v.At) < TTL
}

// For is the same visit, as the account a browser is on now and under that account's name.
func (v Visit) For(account players.AccountID, author players.Author) Visit {
	v.Account = account
	v.Author = author
	return v
}

// Entry is one line of the roster.
type Entry struct {
	Key     Key
	Name    string
	Country string
	Guest   bool
	// Admin is never a guest: a profile with no name does not show as one.
	Admin bool
}

// EntryOf is the line the visit shows on the roster.
func EntryOf(visit Visit) Entry {
	return Entry{
		Key:     visit.Key,
		Name:    visit.Author.Name,
		Country: visit.Country,
		Guest:   visit.Author.Guest,
		Admin:   visit.Author.Admin && !visit.Author.Guest,
	}
}

// Change is one line of the roster that changed: an entry that joined or changed, or, when Left, the entry
// whose key is gone.
type Change struct {
	Entry Entry
	Left  bool
}

// RosterOf is every fresh visit at now: players with a username first, then guests, each group by name
// ignoring case. The key breaks a tie, so the order never changes between two reads.
func RosterOf(visits []Visit, now time.Time) []Entry {
	roster := make([]Entry, 0, len(visits))
	for _, visit := range visits {
		if !visit.Fresh(now) {
			continue
		}
		roster = append(roster, EntryOf(visit))
	}

	slices.SortFunc(roster, func(a, b Entry) int {
		if a.Guest != b.Guest {
			if a.Guest {
				return 1
			}
			return -1
		}
		return cmp.Or(
			strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name)),
			strings.Compare(a.Name, b.Name),
			strings.Compare(string(a.Key), string(b.Key)),
		)
	})

	return roster
}
