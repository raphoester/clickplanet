package presence

import (
	"cmp"
	"errors"
	"slices"
	"strings"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
)

// TTL outlasts a 30s announce on a hidden tab, whose timers may fire only once a minute.
const TTL = 90 * time.Second

var ErrUnknownCountry = errors.New("unknown country")

type Key string

type Visit struct {
	Account players.AccountID
	Key     Key
	Author  players.Author
	Tag     players.Tag
	Country string
	At      time.Time
}

func (v Visit) Fresh(now time.Time) bool {
	return now.Sub(v.At) < TTL
}

func (v Visit) For(account players.AccountID, author players.Author) Visit {
	v.Account = account
	v.Author = author
	return v
}

type Entry struct {
	Key     Key
	Name    string
	Country string
	Guest   bool
	Admin   bool
	Color   players.Color
	Streak  uint32
}

func EntryOf(visit Visit) Entry {
	return Entry{
		Key:     visit.Key,
		Name:    visit.Author.Name,
		Country: visit.Country,
		Guest:   visit.Author.Guest,
		Admin:   visit.Author.Admin && !visit.Author.Guest,
		Color:   visit.Author.Color,
		Streak:  visit.Author.Streak.Days,
	}
}

type Change struct {
	Entry Entry
	Left  bool
}

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
