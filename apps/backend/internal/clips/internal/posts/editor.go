package posts

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
)

// Two lines at most over the globe.
const headlineRunes = 60

// A post the job made.
type Telling struct {
	Key       StoryKey
	Countries []string
	At        time.Time
	Title     string
	State     string
	Failure   string
}

const (
	recent       = 72 * time.Hour
	reheadlining = 7 * 24 * time.Hour
	lately       = 6
)

// Inclusive: the posts at this time or after are what was told.
func ToldSince(now time.Time) time.Time {
	return now.Add(-reheadlining)
}

type Offer struct {
	Count int
	// The seasonal stories told first whatever the editor says, by their number in the findings, from 1.
	First    []int
	Findings Findings
	Recent   []Telling
	Shown    map[string]int
	Now      time.Time
}

func (o Offer) Slots() int {
	return max(o.Count-len(o.First), 0)
}

func (o Offer) Open() bool {
	return o.Slots() > 0 && o.Findings.Any()
}

// A seasonal story is told once; nothing else is kept from the editor.
func (f Findings) Offer(count int, told []Telling, now time.Time) Offer {
	past := pastOf(told, now)
	offer := Offer{Count: count, Findings: f, Shown: map[string]int{}, Now: now}
	for i, seasonal := range f.Seasonal {
		if len(offer.First) < count && !past.keys.Contains(seasonal.Key) {
			offer.First = append(offer.First, i+1)
		}
	}
	for _, flags := range past.shown {
		flags.ForEach(func(flag string) { offer.Shown[flag]++ })
	}
	for _, telling := range told {
		if !telling.At.Before(now.Add(-recent)) {
			offer.Recent = append(offer.Recent, telling)
		}
	}
	slices.SortStableFunc(offer.Recent, func(a, b Telling) int { return b.At.Compare(a.At) })
	return offer
}

type Edition struct {
	Stories []Story
	Why     string
}

// Zero when nothing was sent; USD is estimated from the model's prices.
type Spend struct {
	Model            string
	InputTokens      int64
	CacheWriteTokens int64
	CacheReadTokens  int64
	OutputTokens     int64
	USD              float64
}

type Editorial struct {
	Asked bool
	// Why no answer was had: no key, an API error, a refusal, an answer that cannot be read.
	Failed string
	Why    string
	Spend  Spend
}

type Chosen struct {
	Pick  Pick
	Story Story
	Key   StoryKey
}

type Skipped struct {
	Story Story
	Why   string
}

type Choice struct {
	Chosen  []Chosen
	Skipped []Skipped
	// 0 when every platform's day is full.
	Room      int
	Assigned  []Assignment
	Editorial Editorial
}

func (c Choice) Picks() []Pick {
	picks := make([]Pick, 0, len(c.Chosen))
	for _, chosen := range c.Chosen {
		picks = append(picks, chosen.Pick)
	}
	return picks
}

// The seasonal stories first, then the editor's that hold, one story a key; the rest skipped with why.
func (f Findings) Edited(offer Offer, edition Edition, told []Telling, now time.Time) Choice {
	past := pastOf(told, now)
	choice := Choice{Room: offer.Count}
	tell := func(story Story, key StoryKey) {
		choice.Chosen = append(choice.Chosen, Chosen{Pick: Pick(len(choice.Chosen) + 1), Story: story, Key: key})
	}
	skip := func(story Story, why string) {
		choice.Skipped = append(choice.Skipped, Skipped{Story: story, Why: why})
	}

	for _, number := range offer.First {
		seasonal := f.Seasonal[number-1]
		past.headlines.Add(headlineOf(seasonal.Title))
		tell(Story{Seasonal: number}, seasonal.Key)
	}

	keys := cpcolls.NewSet[StoryKey]()
	for _, story := range edition.Stories {
		if len(choice.Chosen) >= offer.Count {
			skip(story, "the run is full")
			continue
		}
		key, err := f.Check(story)
		if err != nil {
			skip(story, err.Error())
			continue
		}
		if story.Seasonal > 0 {
			skip(story, "a seasonal story is the run's to tell first, not the editor's")
			continue
		}
		headline, err := headlineFrom(story.Headline)
		if err != nil {
			skip(story, err.Error())
			continue
		}
		if past.headlines.Contains(headline) {
			skip(story, fmt.Sprintf("%q was a headline of the last week, or of this run", headline))
			continue
		}
		if keys.Contains(key) {
			skip(story, fmt.Sprintf("%s in %s is told already in this run", key.Flag, key.Continent))
			continue
		}
		keys.Add(key)
		past.headlines.Add(headline)
		story.Headline = headline
		tell(story, key)
	}
	return choice
}

type past struct {
	keys      *cpcolls.Set[StoryKey]
	headlines *cpcolls.Set[string]
	shown     []*cpcolls.Set[string]
}

func pastOf(told []Telling, now time.Time) *past {
	newest := slices.SortedFunc(slices.Values(told), func(a, b Telling) int { return b.At.Compare(a.At) })

	p := &past{keys: cpcolls.NewSet[StoryKey](), headlines: cpcolls.NewSet[string]()}
	for _, telling := range newest {
		if telling.At.Before(ToldSince(now)) {
			continue
		}
		if !telling.At.Before(now.Add(-recent)) {
			p.keys.Add(telling.Key)
		}
		p.headlines.Add(headlineOf(telling.Title))
		if len(p.shown) < lately {
			flags := cpcolls.NewSet(telling.Countries...)
			flags.Add(telling.Key.Flag)
			p.shown = append(p.shown, flags)
		}
	}
	return p
}

func headlineOf(title string) string {
	return strings.ToUpper(strings.TrimSpace(title))
}

var errHeadline = errors.New("the headline cannot be drawn")

// Starts with a letter or a digit, so the generator never reads it as a flag.
func headlineFrom(written string) (string, error) {
	headline := headlineOf(strings.Join(strings.Fields(written), " "))
	if headline == "" || utf8.RuneCountInString(headline) > headlineRunes {
		return "", fmt.Errorf("%w: %q is empty or over %d letters", errHeadline, written, headlineRunes)
	}
	if first, _ := utf8.DecodeRuneInString(headline); !unicode.IsLetter(first) && !unicode.IsDigit(first) {
		return "", fmt.Errorf("%w: %q starts with %q", errHeadline, written, first)
	}
	for _, r := range headline {
		if !unicode.IsLetter(r) && !unicode.IsMark(r) && !unicode.IsDigit(r) && !strings.ContainsRune(" !?'’%:,.&-", r) {
			return "", fmt.Errorf("%w: %q has %q", errHeadline, written, r)
		}
	}
	return headline, nil
}
