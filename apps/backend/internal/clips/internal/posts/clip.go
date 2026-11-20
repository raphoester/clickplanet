package posts

import (
	"errors"
	"fmt"
	"slices"
	"unicode/utf8"
)

type VideoFile string

type PhotoFile string

// The most a TikTok carousel takes.
const maxPhotos = 35

var ErrInvalidClip = errors.New("the clip is invalid")

type TaggedCountry struct {
	Code string
	// With no #: "uk", "christmasisland".
	Tag string
}

// A player a clip names, with the flag it fought for.
type ClipFighter struct {
	Number FighterNumber
	Name   string
	Flag   string
}

type Clip struct {
	pick         Pick
	title        string
	key          StoryKey
	teasers      []string
	countries    []TaggedCountry
	continentTag string
	attribution  string
	video        VideoFile
	seconds      float64
	teaserNames  []string
	record       ClipRecord
	cast         string
	fighters     []ClipFighter
	experiment   ExperimentName
	photos       []PhotoFile
	list         ListName
}

// Photos on TikTok in place of the video, which is their slideshow.
func (c Clip) carousel() bool {
	return len(c.photos) > 0
}

func (c Clip) countryCodes() []string {
	codes := make([]string, 0, len(c.countries))
	for _, country := range c.countries {
		codes = append(codes, country.Code)
	}
	return codes
}

func (c Clip) fighterNumbers() []FighterNumber {
	if len(c.fighters) == 0 {
		return nil
	}
	numbers := make([]FighterNumber, 0, len(c.fighters))
	for _, fighter := range c.fighters {
		numbers = append(numbers, fighter.Number)
	}
	return numbers
}

func (c Clip) TakeSnapshot() ClipSnapshot {
	return ClipSnapshot{
		Pick: c.pick, Title: c.title, Key: c.key, Teasers: c.teasers, Countries: c.countries,
		ContinentTag: c.continentTag, Attribution: c.attribution, Video: c.video, Seconds: c.seconds, TeaserNames: c.teaserNames,
		Record: c.record, Cast: c.cast, Fighters: c.fighters, Experiment: c.experiment, Photos: c.photos, List: c.list,
	}
}

type ClipSnapshot struct {
	Pick  Pick
	Title string
	Key   StoryKey
	// The headline said in other words: the caption opens with one of them.
	Teasers      []string
	Countries    []TaggedCountry
	ContinentTag string
	// What the licence of the clip's music asks the caption to say; empty when none.
	Attribution string
	Video       VideoFile
	// How long the video plays, from the generator's record: 0 for a clip written before it kept one.
	Seconds float64
	// The name of each teaser's wording, in the same order; none for a clip written before they were named.
	TeaserNames []string
	Record      ClipRecord
	// The cast its fighters are numbered in; empty when it names nobody.
	Cast     string
	Fighters []ClipFighter
	// The lab's experiment it was written for; empty for the control.
	Experiment ExperimentName
	// A carousel's photos, in the order they are swiped, and its list; none for a clip.
	Photos []PhotoFile
	List   ListName
}

func (s ClipSnapshot) Restore() (Clip, error) {
	switch {
	case s.Pick < 1:
		return Clip{}, fmt.Errorf("%w: no pick", ErrInvalidClip)
	case s.Title == "":
		return Clip{}, fmt.Errorf("%w: no title", ErrInvalidClip)
	case len(s.Teasers) == 0:
		return Clip{}, fmt.Errorf("%w: no teaser", ErrInvalidClip)
	case !s.Key.known():
		return Clip{}, fmt.Errorf("%w: no story key", ErrInvalidClip)
	case s.Video == "":
		return Clip{}, fmt.Errorf("%w: no video", ErrInvalidClip)
	}
	for _, teaser := range s.Teasers {
		if teaser == "" {
			return Clip{}, fmt.Errorf("%w: an empty teaser", ErrInvalidClip)
		}
	}
	if len(s.TeaserNames) != 0 && len(s.TeaserNames) != len(s.Teasers) {
		return Clip{}, fmt.Errorf("%w: %d teasers and %d names", ErrInvalidClip, len(s.Teasers), len(s.TeaserNames))
	}
	for _, name := range s.TeaserNames {
		if name == "" {
			return Clip{}, fmt.Errorf("%w: a teaser with no name", ErrInvalidClip)
		}
	}
	for _, country := range s.Countries {
		if country.Code == "" || country.Tag == "" {
			return Clip{}, fmt.Errorf("%w: a country with no code or no tag", ErrInvalidClip)
		}
	}
	if len(s.Fighters) > 0 && s.Cast == "" {
		return Clip{}, fmt.Errorf("%w: fighters of no cast", ErrInvalidClip)
	}
	for _, fighter := range s.Fighters {
		if fighter.Number < 1 || fighter.Name == "" || fighter.Flag == "" {
			return Clip{}, fmt.Errorf("%w: a fighter with no number, no name or no flag", ErrInvalidClip)
		}
	}
	if s.Experiment != "" {
		if _, err := ExperimentNameOf(string(s.Experiment)); err != nil {
			return Clip{}, fmt.Errorf("%w: %w", ErrInvalidClip, err)
		}
	}
	if err := s.checkCarousel(); err != nil {
		return Clip{}, err
	}
	return Clip{
		pick: s.Pick, title: s.Title, key: s.Key, teasers: s.Teasers, countries: s.Countries,
		continentTag: s.ContinentTag, attribution: s.Attribution, video: s.Video, seconds: s.Seconds, teaserNames: s.TeaserNames,
		record: s.Record, cast: s.Cast, fighters: s.Fighters, experiment: s.Experiment, photos: s.Photos, list: s.List,
	}, nil
}

func (s ClipSnapshot) checkCarousel() error {
	if len(s.Photos) == 0 && s.List == "" {
		return nil
	}
	switch {
	case !slices.Contains(ListNames, s.List):
		return fmt.Errorf("%w: a carousel of no list %q", ErrInvalidClip, s.List)
	case len(s.Photos) == 0 || len(s.Photos) > maxPhotos:
		return fmt.Errorf("%w: a carousel of %d photos, not 1 to %d", ErrInvalidClip, len(s.Photos), maxPhotos)
	case slices.Contains(s.Photos, ""):
		return fmt.Errorf("%w: a photo with no file", ErrInvalidClip)
	case utf8.RuneCountInString(s.Title) > coverRunes:
		return fmt.Errorf("%w: the title %q is over the %d letters of TikTok's over photos", ErrInvalidClip, s.Title, coverRunes)
	}
	return nil
}
