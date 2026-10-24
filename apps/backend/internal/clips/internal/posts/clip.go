package posts

import (
	"errors"
	"fmt"
)

type VideoFile string

var ErrInvalidClip = errors.New("the clip is invalid")

type TaggedCountry struct {
	Code string
	// With no #: "uk", "christmasisland".
	Tag string
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
	teaserNames  []string
	record       ClipRecord
	variants     Variants
}

func (c Clip) countryCodes() []string {
	codes := make([]string, 0, len(c.countries))
	for _, country := range c.countries {
		codes = append(codes, country.Code)
	}
	return codes
}

func (c Clip) TakeSnapshot() ClipSnapshot {
	return ClipSnapshot{
		Pick: c.pick, Title: c.title, Key: c.key, Teasers: c.teasers, Countries: c.countries,
		ContinentTag: c.continentTag, Attribution: c.attribution, Video: c.video, TeaserNames: c.teaserNames, Record: c.record,
		Variants: c.variants,
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
	// The name of each teaser's wording, in the same order; none for a clip written before they were named.
	TeaserNames []string
	Record      ClipRecord
	// The arm of each experiment the clip is in, as the generator applied or carried it: none when it is in none.
	Variants Variants
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
	return Clip{
		pick: s.Pick, title: s.Title, key: s.Key, teasers: s.Teasers, countries: s.Countries,
		continentTag: s.ContinentTag, attribution: s.Attribution, video: s.Video, teaserNames: s.TeaserNames, record: s.Record,
		variants: s.Variants,
	}, nil
}
