package posts

import (
	"errors"
	"fmt"
	"slices"
	"time"
)

// From 1: a clip's number in its run.
type Pick int

// What a story tells of its flag there, beyond what it did.
type Angle string

const (
	Deeds   Angle = ""
	Holding Angle = "holding"
	DayWon  Angle = "round"
	CallUp  Angle = "callup"
)

// Which story a post told: a post keeps it, so the editor reads what was told.
type StoryKey struct {
	Flag      string
	Continent string
	Angle     Angle
	// The day a DayWon tells or a CallUp follows: "1.5".
	Round string
}

func (k StoryKey) known() bool {
	if k.Flag == "" || k.Continent == "" {
		return false
	}
	switch k.Angle {
	case Deeds, Holding:
		return k.Round == ""
	case DayWon, CallUp:
		return k.Round != ""
	default:
		return false
	}
}

const TheWorld = "the world"

// What the generator found in a replay, measured and never judged. A Story names a front, a tour, a place or a
// seasonal story by its place in its list, from 1.
type Findings struct {
	Since, Until time.Time
	Fronts       []Front
	Tours        []Tour
	Places       []Place
	Seasonal     []Seasonal
	Names        map[string]string
	// Each continent's own flag, empty for none.
	Continents map[string]string
}

type Front struct {
	// The country whose ground alone it is measured on; empty for the whole map.
	Scope        string
	Since, Until time.Time
	TilesChanged int
	// Taken from another flag rather than off empty ground.
	Captured                   int
	Countries                  []FrontCountry
	Flags                      []FrontFlag
	Bombs, Spreads, Enclosures int
	Score                      float64
}

type FrontCountry struct {
	Code                             string
	Continent                        string
	Tiles                            int
	Changed                          int
	HoldersStart, HoldersEnd, Takers map[string]int
}

type FrontFlag struct {
	Code               string
	Continent          string
	Took, Filled, Lost int
	TookFrom           map[string]int
	Solidity           float64
}

type Tour struct {
	Flag                  string
	Since, Until          time.Time
	Took                  int
	ByContinent, TookFrom map[string]int
	// What the second flag across the world took in the same stretch.
	NextTook int
	Solidity float64
	Bombs    int
}

type PlaceKind string

const (
	World          PlaceKind = "world"
	ContinentPlace PlaceKind = "continent"
	CountryPlace   PlaceKind = "country"
)

type Place struct {
	Kind      PlaceKind
	Code      string
	Continent string
	Tiles     int
	Holder    string
	Held      int
	Share     float64
	Rivals    map[string]int
	Changed   int
}

// A day won or a call-up, headlined by the generator.
type Seasonal struct {
	Title     string
	Key       StoryKey
	Countries []string
	Teasers   []string
	Facts     StoryFacts
}

// A clips.v1.StoryFacts in protojson.
type StoryFacts []byte

var ErrInvalidFindings = errors.New("the findings are invalid")

func FindingsOf(findings Findings) (Findings, error) {
	for _, seasonal := range findings.Seasonal {
		if !seasonal.Key.known() || seasonal.Key.Angle == Deeds || seasonal.Key.Angle == Holding {
			return Findings{}, fmt.Errorf("%w: %q is no day won or call-up", ErrInvalidFindings, seasonal.Title)
		}
	}
	for name := range findings.Continents {
		if name == "" {
			return Findings{}, fmt.Errorf("%w: a continent has no name", ErrInvalidFindings)
		}
	}
	return findings, nil
}

func (f Findings) Any() bool {
	return len(f.Fronts)+len(f.Tours)+len(f.Places) > 0
}

type Kind string

const (
	Attack   Kind = "attack"
	Invasion Kind = "invasion"
	Comeback Kind = "comeback"
	Kickout  Kind = "kickout"
	Rout     Kind = "rout"
	Battle   Kind = "battle"
)

var frontKinds = []Kind{Attack, Invasion, Comeback, Kickout, Rout, Battle}

// A story to tell, as the generator's --story takes it. The editor makes one of a front; a tour, a place or a seasonal
// story it only picks.
type Story struct {
	Front, Tour, Place, Seasonal int
	Kind                         Kind
	Attacker, Rival              string
	TakenFrom                    []string
	// One or two countries, else a continent.
	Countries []string
	Continent string
	// The continent whose flags strike back together.
	Team     string
	Headline string
}

func (s Story) Names() string {
	switch {
	case s.Tour > 0:
		return fmt.Sprintf("tour %d", s.Tour)
	case s.Place > 0:
		return fmt.Sprintf("place %d", s.Place)
	case s.Seasonal > 0:
		return fmt.Sprintf("seasonal %d", s.Seasonal)
	default:
		return fmt.Sprintf("front %d", s.Front)
	}
}

var ErrInvalidStory = errors.New("the story does not hold")

// Whether the story is true to the findings, never whether it is good: its key when it is.
func (f Findings) Check(story Story) (StoryKey, error) {
	named := 0
	for _, number := range []int{story.Front, story.Tour, story.Place, story.Seasonal} {
		if number > 0 {
			named++
		}
	}
	if named != 1 {
		return StoryKey{}, fmt.Errorf("%w: it names %d of a front, a tour, a place and a seasonal story, not one", ErrInvalidStory, named)
	}

	switch {
	case story.Seasonal > 0:
		if story.Seasonal > len(f.Seasonal) {
			return StoryKey{}, fmt.Errorf("%w: no seasonal story %d of %d", ErrInvalidStory, story.Seasonal, len(f.Seasonal))
		}
		return f.Seasonal[story.Seasonal-1].Key, nil
	case story.Tour > 0:
		if story.Tour > len(f.Tours) {
			return StoryKey{}, fmt.Errorf("%w: no tour %d of %d", ErrInvalidStory, story.Tour, len(f.Tours))
		}
		return StoryKey{Flag: f.Tours[story.Tour-1].Flag, Continent: TheWorld}, nil
	case story.Place > 0:
		if story.Place > len(f.Places) {
			return StoryKey{}, fmt.Errorf("%w: no place %d of %d", ErrInvalidStory, story.Place, len(f.Places))
		}
		place := f.Places[story.Place-1]
		continent := place.Continent
		if place.Kind == World {
			continent = TheWorld
		}
		return StoryKey{Flag: place.Holder, Continent: continent, Angle: Holding}, nil
	default:
		if story.Front > len(f.Fronts) {
			return StoryKey{}, fmt.Errorf("%w: no front %d of %d", ErrInvalidStory, story.Front, len(f.Fronts))
		}
		return f.checkFront(story, f.Fronts[story.Front-1])
	}
}

func (f Findings) checkFront(story Story, front Front) (StoryKey, error) {
	flagOf := func(code string) (FrontFlag, bool) {
		at := slices.IndexFunc(front.Flags, func(flag FrontFlag) bool { return flag.Code == code })
		if at < 0 {
			return FrontFlag{}, false
		}
		return front.Flags[at], true
	}
	refused := func(why string, args ...any) (StoryKey, error) {
		return StoryKey{}, fmt.Errorf("%w: %s on front %d", ErrInvalidStory, fmt.Sprintf(why, args...), story.Front)
	}

	if !slices.Contains(frontKinds, story.Kind) {
		return refused("%q is no kind a front tells", story.Kind)
	}
	attacker, found := flagOf(story.Attacker)
	if !found || attacker.Took == 0 {
		return refused("%q took nothing", story.Attacker)
	}
	for _, code := range story.TakenFrom {
		if code == story.Attacker {
			return refused("%q took from itself", code)
		}
		if loser, found := flagOf(code); !found || loser.Lost == 0 {
			return refused("%q lost nothing", code)
		}
	}
	switch story.Kind {
	case Battle:
		rival, found := flagOf(story.Rival)
		if !found || story.Rival == story.Attacker || attacker.TookFrom[story.Rival] == 0 || rival.TookFrom[story.Attacker] == 0 {
			return refused("%q and %q did not take from each other", story.Attacker, story.Rival)
		}
	case Rout, Kickout:
		if len(story.TakenFrom) == 0 {
			return refused("a %s names the flag thrown out", story.Kind)
		}
	default:
	}
	if story.Kind != Battle && story.Rival != "" {
		return refused("only a battle has a rival, and this is %s", story.Kind)
	}

	var there []FrontCountry
	switch {
	case len(story.Countries) > 0 && story.Continent == "", len(story.Countries) == 0 && story.Continent != "":
	default:
		return refused("a story is told in one or two countries, or on a continent")
	}
	if len(story.Countries) > 2 {
		return refused("a story is told in one or two countries, not %d", len(story.Countries))
	}
	for _, code := range story.Countries {
		at := slices.IndexFunc(front.Countries, func(country FrontCountry) bool { return country.Code == code })
		if at < 0 {
			return refused("%q is no country of it", code)
		}
		there = append(there, front.Countries[at])
	}
	if story.Continent != "" {
		for _, country := range front.Countries {
			if country.Continent == story.Continent {
				there = append(there, country)
			}
		}
		if len(there) == 0 {
			return refused("%q is no continent of it", story.Continent)
		}
	}
	placeName := story.Continent
	if placeName == "" {
		placeName = fmt.Sprint(story.Countries)
	}
	tookThere := slices.ContainsFunc(there, func(country FrontCountry) bool { return country.Takers[story.Attacker] > 0 })
	if !tookThere {
		return refused("%q took nothing in %s", story.Attacker, placeName)
	}

	switch story.Kind {
	case Comeback:
		home := story.Continent != "" && attacker.Continent == story.Continent || slices.Contains(story.Countries, story.Attacker)
		if !home {
			return refused("%q is not at home in %s, so it is no comeback", story.Attacker, placeName)
		}
	case Kickout:
		heldThere := slices.ContainsFunc(there, func(country FrontCountry) bool { return country.HoldersStart[story.Attacker] > 0 })
		if !heldThere {
			return refused("%q held nothing in %s when the front opened, so it kicks nobody out", story.Attacker, placeName)
		}
	case Rout:
		loser := story.TakenFrom[0]
		heldThere := slices.ContainsFunc(there, func(country FrontCountry) bool { return country.HoldersStart[loser] > 0 })
		if !heldThere {
			return refused("%q held nothing in %s when the front opened, so it is not thrown out", loser, placeName)
		}
	default:
	}
	if story.Team != "" && (story.Kind != Comeback || story.Team != story.Continent || attacker.Continent != story.Team) {
		return refused("a team is a continent's flags striking back together on it, and %q is not that", story.Team)
	}

	continent := story.Continent
	if continent == "" {
		continent = there[0].Continent
	}
	return StoryKey{Flag: f.keyFlagOf(story, attacker), Continent: continent}, nil
}

// The loser of a rout, the continent's flag of a team, else the attacker: the generator's sidesOf.
func (f Findings) keyFlagOf(story Story, attacker FrontFlag) string {
	switch {
	case story.Kind == Rout:
		return story.TakenFrom[0]
	case story.Team != "" && f.Continents[story.Team] != "":
		return f.Continents[story.Team]
	default:
		return attacker.Code
	}
}
