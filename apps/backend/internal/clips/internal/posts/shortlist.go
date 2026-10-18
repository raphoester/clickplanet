package posts

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
)

// From 1: what the generator's --pick takes.
type Pick int

// What a story tells of its flag there, beyond what it did.
type Angle string

const (
	// What the flag did: an attack, a battle, a flag thrown out.
	Deeds Angle = ""
	// How much of the place the flag holds: "CROATIA HOLDS 80% OF EUROPE".
	Holding Angle = "holding"
	// A day of the season the flag won: "FRANCE WINS DAY 5".
	DayWon Angle = "round"
)

// Two stories with one key are one story told twice, whatever their headlines.
type StoryKey struct {
	Flag      string
	Continent string
	Angle     Angle
	// The day a DayWon tells, as its season and its number: "1.5". France winning two days is two stories.
	Round string
}

func (k StoryKey) known() bool {
	return k.Flag != "" && k.Continent != "" && (k.Angle == DayWon) == (k.Round != "")
}

type Story struct {
	Pick  Pick
	Title string
	Key   StoryKey
	// Every country the clip shows, the key's flag among them or not.
	Countries []string
}

func (s Story) flags() *cpcolls.Set[string] {
	return flagsOf(s.Key, s.Countries)
}

type Telling struct {
	Key StoryKey
	// None for a post made before the posts kept them.
	Countries []string
	At        time.Time
	Title     string
	// As the service says it: scheduled, published, failed...
	State string
	// Why a platform did not take it; empty when none refused it.
	Failure string
}

const (
	retelling = 72 * time.Hour
	// Longer than a key: the same headline on another continent reads as the same clip.
	reheadlining = 7 * 24 * time.Hour
	// Half a day of posts at 6 a run.
	lately = 6
	// A third of them: a team of two players is not the feed.
	mostLately = 2
)

// Inclusive: the posts at this time or after are what was told.
func ToldSince(now time.Time) time.Time {
	return now.Add(-reheadlining)
}

type Shortlist []Story

var ErrInvalidStory = errors.New("the story is invalid")

func ShortlistOf(stories ...Story) (Shortlist, error) {
	ranked := make(Shortlist, 0, len(stories))
	for i, story := range stories {
		if !story.Key.known() {
			return nil, fmt.Errorf("%w: %q has no key", ErrInvalidStory, story.Title)
		}
		story.Pick = Pick(i + 1)
		ranked = append(ranked, story)
	}
	return ranked, nil
}

type Choice struct {
	Chosen Shortlist
	// Skipped: its key told in the last three days, its headline in the last week, or either by a story of this run.
	Retold Shortlist
	// Not told, but its flag is in 2 of the last posts, or a story after it was chosen: its flags in fewer of them, or
	// an angle the run had not told.
	Passed Shortlist
}

// A day of the season won goes first, whatever the posts showed: it is the day's news. Then a run tells each angle it
// has once before it tells any twice, the freshest flag first: on Oct 18 a run skipped 8 of its 13 stories as told,
// nearly all one flag in Europe, and a holding is a new story for that flag.
func (s Shortlist) Best(count int, told []Telling, now time.Time) Choice {
	past := pastOf(told, now)
	chosen := cpcolls.NewSet[Pick]()
	angles := cpcolls.NewSet[Angle]()
	var last Pick
	for chosen.Len() < count {
		story, found := s.next(past, angles)
		if !found {
			break
		}
		chosen.Add(story.Pick)
		angles.Add(story.Key.Angle)
		last = max(last, story.Pick)
		past.tell(story)
	}

	var choice Choice
	for _, story := range s {
		switch {
		case chosen.Contains(story.Pick):
			choice.Chosen = append(choice.Chosen, story)
		case !past.untold(story):
			choice.Retold = append(choice.Retold, story)
		case story.Pick < last || past.crowded(story.Key.Flag):
			choice.Passed = append(choice.Passed, story)
		}
	}
	return choice
}

func (s Shortlist) next(past *past, told *cpcolls.Set[Angle]) (Story, bool) {
	for _, story := range s {
		if story.Key.Angle == DayWon && past.untold(story) {
			return story, true
		}
	}
	if story, found := s.freshest(past, func(story Story) bool { return !told.Contains(story.Key.Angle) }); found {
		return story, true
	}
	return s.freshest(past, func(Story) bool { return true })
}

func (s Shortlist) freshest(past *past, among func(Story) bool) (Story, bool) {
	var best Story
	found := false
	for _, story := range s {
		if !among(story) || !past.untold(story) || past.crowded(story.Key.Flag) {
			continue
		}
		if !found || past.fresher(story, best) {
			best, found = story, true
		}
	}
	return best, found
}

func (s Shortlist) Picks() []Pick {
	picks := make([]Pick, 0, len(s))
	for _, story := range s {
		picks = append(picks, story.Pick)
	}
	return picks
}

type past struct {
	keys      *cpcolls.Set[StoryKey]
	headlines *cpcolls.Set[string]
	// The flags each of the last posts showed, newest first.
	shown []*cpcolls.Set[string]
}

func pastOf(told []Telling, now time.Time) *past {
	newest := slices.SortedFunc(slices.Values(told), func(a, b Telling) int { return b.At.Compare(a.At) })

	p := &past{keys: cpcolls.NewSet[StoryKey](), headlines: cpcolls.NewSet[string]()}
	for _, telling := range newest {
		if telling.At.Before(ToldSince(now)) {
			continue
		}
		if !telling.At.Before(now.Add(-retelling)) {
			p.keys.Add(telling.Key)
		}
		p.headlines.Add(headlineOf(telling.Title))
		if len(p.shown) < lately {
			p.shown = append(p.shown, flagsOf(telling.Key, telling.Countries))
		}
	}
	return p
}

func (p *past) tell(story Story) {
	p.keys.Add(story.Key)
	p.headlines.Add(headlineOf(story.Title))
	p.shown = append([]*cpcolls.Set[string]{story.flags()}, p.shown[:min(len(p.shown), lately-1)]...)
}

func (p *past) untold(story Story) bool {
	return !p.keys.Contains(story.Key) && !p.headlines.Contains(headlineOf(story.Title))
}

func (p *past) crowded(flag string) bool {
	return p.mentions(flag) >= mostLately
}

func (p *past) mentions(flag string) int {
	count := 0
	for _, flags := range p.shown {
		if flags.Contains(flag) {
			count++
		}
	}
	return count
}

// Its own flag first, then the flag of it the last posts showed most.
func (p *past) fresher(story, than Story) bool {
	if own, other := p.mentions(story.Key.Flag), p.mentions(than.Key.Flag); own != other {
		return own < other
	}
	return p.staleness(story) < p.staleness(than)
}

func (p *past) staleness(story Story) int {
	most := 0
	story.flags().ForEach(func(flag string) {
		most = max(most, p.mentions(flag))
	})
	return most
}

func flagsOf(key StoryKey, countries []string) *cpcolls.Set[string] {
	flags := cpcolls.NewSet(countries...)
	flags.Add(key.Flag)
	return flags
}

func headlineOf(title string) string {
	return strings.ToUpper(strings.TrimSpace(title))
}
