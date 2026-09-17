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
	"unicode"
	"unicode/utf8"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
)

const (
	// TTL is how long a visit counts after its announce. A client announces every 30s, and a hidden tab's timers
	// may fire once a minute, so a live player is never dropped between two announces.
	TTL = 90 * time.Second
	// MaxGuestNameLength bounds a guest's name in runes, before the prefix, as the chat does.
	MaxGuestNameLength = 24
	// GuestPrefix goes before every guest's name, as in the chat. No username starts with it.
	GuestPrefix = players.ReservedPrefix
)

var ErrUnknownCountry = errors.New("unknown country")

// Visit is a player's last announce.
type Visit struct {
	Account players.AccountID
	// Username is empty for an account that chose none: a guest.
	Username  players.Name
	GuestName string
	Tag       players.Tag
	Country   string
	At        time.Time
}

// Fresh is whether the visit still counts at now.
func (v Visit) Fresh(now time.Time) bool {
	return now.Sub(v.At) < TTL
}

// For is the same visit, as the account a browser is on now and under that account's username.
func (v Visit) For(account players.AccountID, username players.Name) Visit {
	v.Account = account
	v.Username = username
	return v
}

func (v Visit) guest() bool {
	return v.Username == ""
}

// displayName is the username, or the prefix and the guest's name, or the prefix and the tag.
func (v Visit) displayName() string {
	switch {
	case !v.guest():
		return string(v.Username)
	case v.GuestName != "":
		return GuestPrefix + v.GuestName
	default:
		return GuestPrefix + string(v.Tag)
	}
}

// GuestNameOf is the name a guest typed, cleaned as the chat cleans it: tabs become spaces, control characters
// go, and the ends are trimmed. A name that is not valid UTF-8, or longer than MaxGuestNameLength, is empty:
// the guest is then shown by its tag.
func GuestNameOf(value string) string {
	if !utf8.ValidString(value) {
		return ""
	}

	var b strings.Builder
	for _, r := range value {
		if r == '\t' {
			r = ' '
		}
		if !unicode.IsControl(r) {
			b.WriteRune(r)
		}
	}

	name := strings.TrimSpace(b.String())
	if utf8.RuneCountInString(name) > MaxGuestNameLength {
		return ""
	}
	return name
}

// Entry is one line of the roster.
type Entry struct {
	Name    string
	Tag     players.Tag
	Country string
	Guest   bool
}

// RosterOf is every fresh visit at now: players with a username first, then guests, each group by name
// ignoring case. The tag breaks a tie, so two guests of one name keep their order between two reads.
func RosterOf(visits []Visit, now time.Time) []Entry {
	roster := make([]Entry, 0, len(visits))
	for _, visit := range visits {
		if !visit.Fresh(now) {
			continue
		}
		roster = append(roster, Entry{
			Name:    visit.displayName(),
			Tag:     visit.Tag,
			Country: visit.Country,
			Guest:   visit.guest(),
		})
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
			strings.Compare(string(a.Tag), string(b.Tag)),
		)
	})

	return roster
}
