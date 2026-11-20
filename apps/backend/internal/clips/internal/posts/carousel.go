package posts

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"
	"unicode"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
)

// A carousel's list (docs/video-lab.md, "Photos, not a video"): what its photos rank.
type ListName string

const (
	FightTheMost   ListName = "fight-the-most"
	FarFromHome    ListName = "far-from-home"
	BiggestEmpires ListName = "biggest-empires"
)

var ListNames = []ListName{FightTheMost, FarFromHome, BiggestEmpires}

// A list's photos between the cover and the last.
const CarouselRanks = 5

const (
	// TikTok's title over the photos.
	coverRunes = 90
	rankRunes  = 40
)

// What a story of a carousel tells of its flags: the 1st of a list.
const Listed Angle = "list"

type Lists struct {
	FightTheMost   []FightEntry
	FarFromHome    []FarLandEntry
	BiggestEmpires []EmpireEntry
	FoughtSince    time.Time
}

type FightEntry struct {
	Flag       string
	Fought     int
	Took       int
	Lost       int
	Rival      string
	RivalTiles int
	Country    string
}

type FarLandEntry struct {
	Flag    string
	Country string
	Tiles   int
	Km      int
}

type EmpireEntry struct {
	Flag    string
	Tiles   int
	Share   float64
	Country string
}

type Rank struct {
	Code string
	Line string
}

var ErrUnknownList = errors.New("no list is named so")

// The flag of each entry, the top first.
func (l Lists) codesOf(list ListName) ([]string, error) {
	switch list {
	case FightTheMost:
		return codes(l.FightTheMost, func(entry FightEntry) string { return entry.Flag }), nil
	case FarFromHome:
		return codes(l.FarFromHome, func(entry FarLandEntry) string { return entry.Flag }), nil
	case BiggestEmpires:
		return codes(l.BiggestEmpires, func(entry EmpireEntry) string { return entry.Flag }), nil
	default:
		return nil, fmt.Errorf("%w: %q", ErrUnknownList, list)
	}
}

func codes[T any](entries []T, codeOf func(T) string) []string {
	written := make([]string, 0, len(entries))
	for _, entry := range entries {
		written = append(written, codeOf(entry))
	}
	return written
}

// A list told with the same 5 as its last post is not told again.
func (f Findings) carouselTellable(story Story, told []Telling, keys *cpcolls.Set[StoryKey], experiment ExperimentName) (Story, StoryKey, error) {
	refused := func(why string, args ...any) (Story, StoryKey, error) {
		return Story{}, StoryKey{}, fmt.Errorf("%w: %s", ErrInvalidStory, fmt.Sprintf(why, args...))
	}

	entries, err := f.Lists.codesOf(story.List)
	if err != nil {
		return refused("%s", err)
	}
	if len(entries) < CarouselRanks {
		return refused("%s has %d entries, not %d", story.List, len(entries), CarouselRanks)
	}
	if len(story.Ranks) != CarouselRanks {
		return refused("a carousel tells %d ranks, not %d", CarouselRanks, len(story.Ranks))
	}
	ranks := make([]Rank, 0, CarouselRanks)
	for i, rank := range story.Ranks {
		if rank.Code != entries[i] {
			return refused("rank %d of %s is %q, not %q", i+1, story.List, entries[i], rank.Code)
		}
		line, err := rankLineFrom(rank.Line)
		if err != nil {
			return Story{}, StoryKey{}, err
		}
		ranks = append(ranks, Rank{Code: rank.Code, Line: line})
	}
	cover, err := drawn("cover", headlineOf(oneLine(story.Hook)), coverRunes)
	if err != nil {
		return Story{}, StoryKey{}, err
	}
	shown := entries[:CarouselRanks]
	if last, found := lastTold(told, story.List); found && sameCodes(last.Ranks, shown) {
		return refused("the 5 of %s are the 5 of its last post, at %s", story.List, last.At.UTC().Format(time.RFC3339))
	}

	key := StoryKey{Flag: shown[0], Continent: TheWorld, Angle: Listed}
	if keys.Contains(key) {
		return refused("%s in %s is told already in this run", key.Flag, key.Continent)
	}
	keys.Add(key)
	return Story{List: story.List, Ranks: ranks, Hook: cover, Experiment: experiment, Language: English}, key, nil
}

// The number on the photo is the generator's, so the words hold none.
func rankLineFrom(written string) (string, error) {
	line, err := drawn("rank's line", headlineOf(oneLine(written)), rankRunes)
	if err != nil {
		return "", err
	}
	if strings.ContainsFunc(line, unicode.IsDigit) {
		return "", fmt.Errorf("%w: the rank's line %q has a number: the photo draws the list's", errUndrawable, line)
	}
	return line, nil
}

func lastTold(told []Telling, list ListName) (Telling, bool) {
	var last Telling
	found := false
	for _, telling := range told {
		if telling.List == list && (!found || telling.At.After(last.At)) {
			last, found = telling, true
		}
	}
	return last, found
}

func sameCodes(a, b []string) bool {
	return slices.Equal(slices.Sorted(slices.Values(a)), slices.Sorted(slices.Values(b)))
}
