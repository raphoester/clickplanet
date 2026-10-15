package posts

import (
	"errors"
	"fmt"
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

const retelling = 24 * time.Hour

// Inclusive: a story told at this time or after is not told again.
func ToldSince(now time.Time) time.Time {
	return now.Add(-retelling)
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
	// Skipped: told in the last day, or by a better story of this run.
	Retold Shortlist
}

func (s Shortlist) Best(count int, told []Telling, now time.Time) Choice {
	keys := cpcolls.NewSet[StoryKey]()
	for _, telling := range told {
		if !telling.At.Before(ToldSince(now)) {
			keys.Add(telling.Key)
		}
	}

	var choice Choice
	for _, story := range s {
		switch {
		case len(choice.Chosen) >= count:
			return choice
		case keys.Contains(story.Key):
			choice.Retold = append(choice.Retold, story)
		default:
			choice.Chosen = append(choice.Chosen, story)
			keys.Add(story.Key)
		}
	}
	return choice
}

func (s Shortlist) Picks() []Pick {
	picks := make([]Pick, 0, len(s))
	for _, story := range s {
		picks = append(picks, story.Pick)
	}
	return picks
}
