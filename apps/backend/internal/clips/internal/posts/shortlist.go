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

// Two stories with one key are one story told twice, whatever their headlines.
type StoryKey struct {
	Flag      string
	Continent string
}

func (k StoryKey) known() bool {
	return k.Flag != "" && k.Continent != ""
}

type Story struct {
	Pick  Pick
	Title string
	Key   StoryKey
}

type Telling struct {
	Key   StoryKey
	At    time.Time
	Title string
	// As the service says it: scheduled, published, failed...
	State string
	// Why a platform did not take it; empty when none refused it.
	Failure string
}

const (
	retelling = 72 * time.Hour
	// Longer than a key: the same headline on another continent reads as the same clip.
	reheadlining = 7 * 24 * time.Hour
	// A day of posts.
	lately = 4
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
	// Better than a story chosen, but of a flag the last posts told more.
	Passed Shortlist
}

func (s Shortlist) Best(count int, told []Telling, now time.Time) Choice {
	past := pastOf(told, now)
	chosen := cpcolls.NewSet[Pick]()
	var last Pick
	for chosen.Len() < count {
		story, found := s.freshest(past)
		if !found {
			break
		}
		chosen.Add(story.Pick)
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
		case story.Pick < last:
			choice.Passed = append(choice.Passed, story)
		}
	}
	return choice
}

func (s Shortlist) freshest(past *past) (Story, bool) {
	var best Story
	found := false
	for _, story := range s {
		if !past.untold(story) {
			continue
		}
		if !found || past.mentions(story.Key.Flag) < past.mentions(best.Key.Flag) {
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
	flags     []string
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
		if len(p.flags) < lately {
			p.flags = append(p.flags, telling.Key.Flag)
		}
	}
	return p
}

func (p *past) tell(story Story) {
	p.keys.Add(story.Key)
	p.headlines.Add(headlineOf(story.Title))
	p.flags = append([]string{story.Key.Flag}, p.flags[:min(len(p.flags), lately-1)]...)
}

func (p *past) untold(story Story) bool {
	return !p.keys.Contains(story.Key) && !p.headlines.Contains(headlineOf(story.Title))
}

func (p *past) mentions(flag string) int {
	count := 0
	for _, told := range p.flags {
		if told == flag {
			count++
		}
	}
	return count
}

func headlineOf(title string) string {
	return strings.ToUpper(strings.TrimSpace(title))
}
