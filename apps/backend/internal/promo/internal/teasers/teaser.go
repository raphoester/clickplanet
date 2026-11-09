// Package teasers is each battle's teaser as the game posts it on its own TikTok and YouTube, about a day before the
// battle starts: when it is due, its words, and the post the publishing module makes of it. The teasers job renders
// the cards, which needs a browser; promo decides the rest.
package teasers

import "time"

// What GetTeaser takes: the battle's entry's id, or a season's finale's round (season-1/finale).
type BattleID string

// A card as players read it, its words filled.
type Card struct {
	BackgroundURL string
	Headline      string
	// May be empty.
	Line string
}

// A battle to come with its cards, as ListTeasers lists it.
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
	// How far ahead a battle's teaser is due: the job runs every 6 hours, so it sees each battle first 30 to 36 hours
	// before it starts.
	Ahead = 36 * time.Hour
	// A teaser goes out this long before its battle at the soonest: about a day before, in TikTok's hours of that day.
	lead = 30 * time.Hour
	// And this long before it at the latest: later, it tells nobody in time to come.
	margin = time.Hour
	// The posts read for the battles teased already: a teaser goes out no sooner than lead before a battle still to
	// come, and twice that covers a battle the schedule moved later since.
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
