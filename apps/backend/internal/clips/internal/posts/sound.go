package posts

import (
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"
)

// The experiment on TikTok's sound: the clip's own, its anthem, or a track trending on TikTok attached over it, the
// anthem muted. Its coin is a clip's, and its arm acts when the clip is posted, on TikTok alone: the video is the same.
const (
	SoundExperiment = "sound"
	// The video's own sound, exactly as before the experiment: the control.
	WithAnthem = "anthem"
	// A track of TikTok's trending chart over the video, the anthem muted.
	WithTrending = "trending"
)

// TikTok's trending chart lists 100: the top of it is what a viewer hears everywhere now, and 20 is wide enough that
// the arm is not one song's. Counted among the tracks that play the whole clip: the chart ranks sound effects of a
// second too. The rank is kept with the post, so the analysis can tell whether a higher one did better.
const trendingTop = 20

// The genres the game plays, of TikTok's tags on the chart's tracks: a track is in when every tag of it is one of them.
// A tag the set does not name is out, so a genre new to the chart is let in here. ALL is the library's effects and beds.
var genresOfTheGame = map[string]bool{
	"ELECTRONIC": true, "HIP_HOP/RAP": true, "ROCK": true, "TECHNO": true, "TRANCE": true, "POP_ROCK": true,
}

var (
	ErrNoTrendingTrack = errors.New("no trending track to attach")
	ErrChartCountry    = errors.New("a chart's country is two letters, ISO 3166-1")
)

// A track of TikTok's trending chart, as the posting service lists it.
type Track struct {
	// What TikTok takes as the sound: the id of its trending excerpt, or of the whole track when it has none (Excerpt
	// false). Empty when the chart gave neither: such a track is never attached.
	ID      string
	Excerpt bool
	Title   string
	Artist  string
	// From 1. 0 when the chart did not say: its place in the list is its rank then.
	Rank int
	// How long what ID names plays; 0 when the chart did not say.
	Seconds int
	// TikTok's tags on it, in upper case ("ELECTRONIC", "HIP_HOP/RAP"); none when the chart gave none.
	Genres []string
}

// A track attached over a clip on TikTok, and how it is mixed: volumes from 0 to 100.
type Sound struct {
	Track Track
	// ISO 3166-1 alpha-2, in upper case: the chart the track was picked from.
	Country        string
	TrackVolume    int
	OriginalVolume int
}

// How the arm `trending` picks and mixes a track.
type Soundtracks struct {
	// The chart read: the country whose TikTok views the clips aim at.
	Country string
}

// Upper case, or an error for anything but two letters.
func ChartCountryOf(code string) (string, error) {
	code = strings.ToUpper(strings.TrimSpace(code))
	if len(code) != 2 || strings.Trim(code, "ABCDEFGHIJKLMNOPQRSTUVWXYZ") != "" {
		return "", fmt.Errorf("%w: %q", ErrChartCountry, code)
	}
	return code, nil
}

// One of the top of the chart that play the whole clip in the game's genres, at random, the chart's own order kept
// (rank 1 first), mixed alone over a muted video: the anthem at 0 rather than the API's default, so nothing rests on a
// default that may change, and the track at 100, since it is all a viewer hears, and a quieter track would put a
// softer clip against the anthem's.
func (s Soundtracks) Pick(chart []Track, clip Clip, dice Dice) (Sound, error) {
	if clip.seconds <= 0 {
		return Sound{}, fmt.Errorf("%w: how long the clip plays is unknown", ErrNoTrendingTrack)
	}
	var top []Track
	for place, track := range chart {
		if track.ID == "" || !track.plays(clip) || !track.fits() {
			continue
		}
		if track.Rank <= 0 {
			track.Rank = place + 1
		}
		top = append(top, track)
		if len(top) == trendingTop {
			break
		}
	}
	if len(top) == 0 {
		return Sound{}, fmt.Errorf("%w: the %s chart lists none in the game's genres that plays the clip's %.1fs",
			ErrNoTrendingTrack, s.Country, clip.seconds)
	}
	return Sound{Track: top[dice.IntN(len(top))], Country: s.Country, TrackVolume: 100, OriginalVolume: 0}, nil
}

// Whether what the track's ID names plays to the end of the clip: TikTok does not loop a track, and the anthem is muted
// under it, so a shorter one leaves the rest of the clip silent. A length the chart did not give is no length.
func (t Track) plays(clip Clip) bool {
	return t.Seconds > 0 && float64(t.Seconds) >= clip.seconds
}

// Whether the track is in the game's genres: tagged, and with no tag beside them.
func (t Track) fits() bool {
	if len(t.Genres) == 0 {
		return false
	}
	for _, genre := range t.Genres {
		if !genresOfTheGame[genre] {
			return false
		}
	}
	return true
}

// Whether the post is to carry a trending track: drawn for one, and going to TikTok.
func (p Post) WantsTrendingSound() bool {
	return p.clip.variants[SoundExperiment] == WithTrending && slices.Contains(p.platforms, TikTok)
}

// The post with the track attached on TikTok.
func (p Post) WithSound(sound Sound) Post {
	p.sound = &sound
	return p
}

// The post as the control sends it, in the experiment's arm of weight 0, and why: drawn for an arm that could not be
// applied, it is still counted, apart.
func (p Post) FellBack(experiment string, why error) Post {
	moved := maps.Clone(p.moved)
	if moved == nil {
		moved = Variants{}
	}
	moved[experiment] = FellBack
	p.moved, p.sound, p.missed = moved, nil, why
	return p
}

// The arm of each experiment the post goes out in: its clip's, and those it was moved to.
func (p Post) variants() Variants {
	if len(p.moved) == 0 {
		return p.clip.variants
	}
	variants := maps.Clone(p.clip.variants)
	if variants == nil {
		variants = Variants{}
	}
	maps.Copy(variants, p.moved)
	return variants
}
