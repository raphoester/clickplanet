// Package teasers is the teasers job's: each battle to come posted about a day before it starts, its cards as a photo
// carousel on TikTok and a slideshow on YouTube, through the server's publishing module. The server's promo module
// knows nothing of it: the job asks promo's ListTeasers and the publishing module over ssh, as an operator does.
package teasers

import (
	"regexp"
	"slices"
	"time"
)

// What GetTeaser takes: the battle's entry's id, or a season's finale's round (season-1/finale).
type BattleID string

// A card as players read it, its words filled.
type Card struct {
	BackgroundURL string
	Headline      string
	// May be empty.
	Line string
}

// A battle to come with its cards, as promo's ListTeasers answers it.
type Teaser struct {
	Battle   BattleID
	StartsAt time.Time
	EndsAt   time.Time
	// The schedule's words for the players. Empty when the operator wrote none.
	Description string
	// How many times a day's points the battle scores.
	Times      uint32
	EndsSeason bool
	// In the operator's order, at least one.
	Cards []Card
}

const (
	// How far ahead a run lists the battles: one that starts later is a later run's. A run every 6 hours sees each
	// battle first 30 to 36 hours before it starts.
	Ahead = 36 * time.Hour
	// A teaser goes out this long before its battle at the soonest: about a day before, in TikTok's hours of that day.
	lead = 30 * time.Hour
	// And this long before it at the latest: later, it tells nobody in time to come.
	margin = time.Hour
	// The posts a run reads for the battles teased already: a teaser goes out no sooner than lead before a battle still
	// to come, and twice that covers a battle the schedule moved later since.
	teasedFor = 2 * lead
)

// When the teaser may go out: from about a day before the battle to an hour before it.
func (t Teaser) Window() (notBefore, by time.Time) {
	return t.StartsAt.Add(-lead), t.StartsAt.Add(-margin)
}

// The posts that say which battles were teased: those since then.
func TeasedSince(now time.Time) time.Time {
	return now.Add(-teasedFor)
}

// The battles listed that no post teases yet, by start as listed. A battle the schedule moved after its teaser went
// out is teased already: its post says the old time, and a second post would be the spam the calendar is there to stop.
func Due(listed []Teaser, teased []BattleID) []Teaser {
	due := make([]Teaser, 0, len(listed))
	for _, teaser := range listed {
		if !slices.Contains(teased, teaser.Battle) {
			due = append(due, teaser)
		}
	}
	return due
}

var unsafe = regexp.MustCompile(`[^A-Za-z0-9._-]`)

// The name of the teaser's folder in a run: its id, with a mark that is no letter, digit, dot, dash or underscore an
// underscore (season-1/finale is season-1_finale). The folder keeps the id itself in its teaser.json.
func (t Teaser) FolderName() string {
	name := unsafe.ReplaceAllString(string(t.Battle), "_")
	if name == "" || name == "." || name == ".." {
		return "_" + name
	}
	return name
}
