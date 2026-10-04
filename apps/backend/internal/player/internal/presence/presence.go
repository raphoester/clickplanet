package presence

import (
	"cmp"
	"errors"
	"slices"
	"strings"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/titles"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/wearing"
)

// TTL outlasts a 30s announce on a hidden tab, whose timers may fire only once a minute.
const TTL = 90 * time.Second

var ErrUnknownCountry = errors.New("unknown country")

type Key string

type Visit struct {
	account players.AccountID
	key     Key
	author  wearing.Author
	tag     players.Tag
	country string
	at      time.Time
}

func NewVisit(account players.AccountID, author wearing.Author, tag players.Tag, country string, at time.Time) Visit {
	return Visit{account: account, author: author, tag: tag, country: country, at: at}
}

func (v Visit) Account() players.AccountID { return v.account }

func (v Visit) Key() Key { return v.key }

func (v Visit) Author() wearing.Author { return v.author }

func (v Visit) Tag() players.Tag { return v.tag }

func (v Visit) At() time.Time { return v.at }

func (v Visit) Fresh(now time.Time) bool {
	return now.Sub(v.at) < TTL
}

func (v Visit) Keyed(key Key) Visit {
	v.key = key
	return v
}

func (v Visit) For(account players.AccountID, author wearing.Author) Visit {
	v.account = account
	v.author = author
	return v
}

type Entry struct {
	key     Key
	name    string
	country string
	guest   bool
	admin   bool
	color   players.Color
	streak  uint32
	title   titles.Standing
}

func EntryOf(visit Visit) Entry {
	return Entry{
		key:     visit.key,
		name:    visit.author.Name(),
		country: visit.country,
		guest:   visit.author.Guest(),
		admin:   visit.author.Admin() && !visit.author.Guest(),
		color:   visit.author.Color(),
		streak:  visit.author.Streak().Days(),
		title:   visit.author.Worn(),
	}
}

func (e Entry) Key() Key { return e.key }

func (e Entry) Name() string { return e.name }

func (e Entry) Country() string { return e.country }

func (e Entry) Guest() bool { return e.guest }

func (e Entry) Admin() bool { return e.admin }

func (e Entry) Color() players.Color { return e.color }

func (e Entry) Streak() uint32 { return e.streak }

func (e Entry) Title() titles.Standing { return e.title }

type Change struct {
	entry Entry
	left  bool
}

func ChangeOf(entry Entry) Change { return Change{entry: entry} }

func DepartureOf(entry Entry) Change { return Change{entry: entry, left: true} }

func (c Change) Entry() Entry { return c.entry }

func (c Change) Left() bool { return c.left }

func RosterOf(visits []Visit, now time.Time) []Entry {
	roster := make([]Entry, 0, len(visits))
	for _, visit := range visits {
		if !visit.Fresh(now) {
			continue
		}
		roster = append(roster, EntryOf(visit))
	}

	slices.SortFunc(roster, func(a, b Entry) int {
		if a.guest != b.guest {
			if a.guest {
				return 1
			}
			return -1
		}
		return cmp.Or(
			strings.Compare(strings.ToLower(a.name), strings.ToLower(b.name)),
			strings.Compare(a.name, b.name),
			strings.Compare(string(a.key), string(b.key)),
		)
	})

	return roster
}
