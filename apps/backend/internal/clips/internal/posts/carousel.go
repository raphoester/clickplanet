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
	OwnLand        ListName = "own-land"
	MostAttacked   ListName = "most-attacked"
	FarFromHome    ListName = "far-from-home"
	BiggestEmpires ListName = "biggest-empires"
)

var ListNames = []ListName{OwnLand, MostAttacked, FarFromHome, BiggestEmpires}

// A list's photos between the cover and the last.
const CarouselRanks = 5

const (
	// TikTok's title over the photos.
	coverRunes = 90
	rankRunes  = 40
)

// What a story of a carousel tells of its flags: the 1st of a list.
const Listed Angle = "list"

// The four lists the generator ranked, the top of each first.
type Lists struct {
	OwnLand        []OwnLandEntry
	MostAttacked   []AttackedEntry
	FarFromHome    []FarLandEntry
	BiggestEmpires []EmpireEntry
	AttackedSince  time.Time
}

type OwnLandEntry struct {
	Country     string
	Tiles       int
	OwnShare    float64
	Holder      string
	HolderShare float64
}

type AttackedEntry struct {
	Flag         string
	Lost         int
	TakenBy      string
	TakenByTiles int
	Country      string
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

// A photo of a carousel: an entry of its list, and the words under its name.
type Rank struct {
	Code string
	Line string
}

var ErrUnknownList = errors.New("no list is named so")

// The code of each entry, the top first: the country of own-land, the flag of the others.
func (l Lists) codesOf(list ListName) ([]string, error) {
	switch list {
	case OwnLand:
		return codes(l.OwnLand, func(entry OwnLandEntry) string { return entry.Country }), nil
	case MostAttacked:
		return codes(l.MostAttacked, func(entry AttackedEntry) string { return entry.Flag }), nil
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

// A carousel's story when it is true to its list: the list's first 5, in its order, each with its words, under a cover
// that can be drawn. A list told with the same 5 as its last post is not told again.
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

// None is no line: the number on the photo is the generator's, so the words say none.
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

// The same codes, in any order.
func sameCodes(a, b []string) bool {
	return slices.Equal(slices.Sorted(slices.Values(a)), slices.Sorted(slices.Values(b)))
}
