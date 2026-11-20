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

const (
	// Two lines at most over the globe.
	headlineRunes = 60
	// Two lines at most at the end, in its big letters.
	callRunes = 30
	lineRunes = 60
)

// A post the job made.
type Telling struct {
	Key       StoryKey
	Countries []string
	At        time.Time
	Title     string
	State     string
	Failure   string
	// What its clip told, from the record kept with it: zero for a post made before the record kept it.
	Facts TellingFacts
}

// What a post's clip told: its kind, its sides, where (countries, else a continent or TheWorld), its line, the stretch it
// played, and the share of the place its key's flag held when it started and when it ended.
type TellingFacts struct {
	Kind                     string
	Attacker, Rival          string
	Countries                []string
	Region                   string
	Line                     string
	Since, Until             time.Time
	ShareAtStart, ShareAtEnd float64
}

const (
	recent       = 72 * time.Hour
	reheadlining = 7 * 24 * time.Hour
	lately       = 6
	// About a season: what its viewers already know, so each post tells what comes next.
	seasonMemory = 28 * 24 * time.Hour
)

// Inclusive: the posts at this time or after are what was told.
func ToldSince(now time.Time) time.Time {
	return now.Add(-seasonMemory)
}

type Offer struct {
	Count int
	// The seasonal stories told first whatever the editor says, by their number in the findings, from 1.
	First    []int
	Findings Findings
	// The posts of the season, newest first: from the season's start when the findings know it, else of the last 3 days.
	Told  []Telling
	Shown map[string]int
	Now   time.Time
	// The pick the lab's experiment fills, after the seasonal stories; none when no experiment is on this run.
	Lab LabSlot
}

func (o Offer) Slots() int {
	return max(o.Count-len(o.First), 0)
}

func (o Offer) Open() bool {
	return o.Slots() > 0 && o.Findings.Any()
}

// A seasonal story is told once; nothing else is kept from the editor. lab: the slot the run's coin gave the lab, which
// moves past the seasonal stories told first.
func (f Findings) Offer(count int, told []Telling, now time.Time, lab LabSlot) Offer {
	past := pastOf(told, now)
	offer := Offer{Count: count, Findings: f, Shown: map[string]int{}, Now: now}
	for i, seasonal := range f.Seasonal {
		if len(offer.First) < count && !past.keys.Contains(seasonal.Key) {
			offer.First = append(offer.First, i+1)
		}
	}
	offer.Lab = lab.after(len(offer.First), count)
	for _, flags := range past.shown {
		flags.ForEach(func(flag string) { offer.Shown[flag]++ })
	}
	since := f.seasonSince(now.Add(-recent))
	if earliest := now.Add(-seasonMemory); since.Before(earliest) {
		since = earliest
	}
	for _, telling := range told {
		if !telling.At.Before(since) {
			offer.Told = append(offer.Told, telling)
		}
	}
	slices.SortStableFunc(offer.Told, func(a, b Telling) int { return b.At.Compare(a.At) })
	return offer
}

// When the history the sagas are measured on starts: the season's start.
func (f Findings) seasonSince(otherwise time.Time) time.Time {
	if len(f.Sagas) == 0 {
		return otherwise
	}
	return f.Sagas[0].Since
}

type Edition struct {
	Stories []Story
	// The story written to the lab's brief, when the run asked for one: the first that holds is told.
	Lab []Story
	Why string
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
	Editorial Editorial
	// The slot the lab was given this run, and whether its story was told there; one that was not left its slot to the
	// control, and its story is in Skipped with why.
	Lab     LabSlot
	LabTold bool
}

func (c Choice) Picks() []Pick {
	picks := make([]Pick, 0, len(c.Chosen))
	for _, chosen := range c.Chosen {
		picks = append(picks, chosen.Pick)
	}
	return picks
}

// The seasonal stories first, then the editor's that hold, one story a key, with the lab's story at its slot; the rest
// skipped with why. A lab story that does not hold leaves its slot to the control's next one.
func (f Findings) Edited(offer Offer, edition Edition, told []Telling, now time.Time) Choice {
	past := pastOf(told, now)
	choice := Choice{Room: offer.Count, Lab: offer.Lab}
	var chosen []Chosen
	skip := func(story Story, why string) {
		choice.Skipped = append(choice.Skipped, Skipped{Story: story, Why: why})
	}

	for _, number := range offer.First {
		seasonal := f.Seasonal[number-1]
		past.headlines.Add(headlineOf(seasonal.Title))
		chosen = append(chosen, Chosen{Story: Story{Seasonal: number}, Key: seasonal.Key})
	}

	keys := cpcolls.NewSet[StoryKey]()
	var lab *Chosen
	for _, story := range edition.Lab {
		switch {
		case !offer.Lab.On():
			skip(story, "the run gave the lab no slot")
		case lab != nil:
			skip(story, "the lab tells one story a run")
		default:
			told, key, err := f.tellable(story, past, keys, offer.Lab.Experiment.Name)
			if err != nil {
				skip(story, fmt.Sprintf("the lab's %s: %s", offer.Lab.Experiment.Name, err))
				continue
			}
			lab = &Chosen{Story: told, Key: key}
		}
	}
	choice.LabTold = lab != nil

	room := offer.Count - len(chosen)
	if lab != nil {
		room--
	}
	var others []Chosen
	for _, story := range edition.Stories {
		if len(others) >= room {
			skip(story, "the run is full")
			continue
		}
		told, key, err := f.tellable(story, past, keys, "")
		if err != nil {
			skip(story, err.Error())
			continue
		}
		others = append(others, Chosen{Story: told, Key: key})
	}

	// The lab's story at its slot, or right after the run's last when the run has fewer.
	for len(others) > 0 || lab != nil {
		if lab != nil && (len(chosen)+1 >= offer.Lab.At || len(others) == 0) {
			chosen = append(chosen, *lab)
			lab = nil
			continue
		}
		chosen, others = append(chosen, others[0]), others[1:]
	}
	for i := range chosen {
		chosen[i].Pick = Pick(i + 1)
	}
	choice.Chosen = chosen
	return choice
}

// The story as it is told, and its key, when it holds; why not, else. The control speaks English and opens on no hook:
// an experiment's story alone has its own (experiment, empty for the control).
func (f Findings) tellable(story Story, past *past, keys *cpcolls.Set[StoryKey], experiment ExperimentName) (Story, StoryKey, error) {
	key, err := f.Check(story)
	if err != nil {
		return Story{}, StoryKey{}, err
	}
	if story.Seasonal > 0 {
		return Story{}, StoryKey{}, errors.New("a seasonal story is the run's to tell first, not the editor's")
	}
	headline, err := headlineFrom(story.Headline)
	if err != nil {
		return Story{}, StoryKey{}, err
	}
	teasers, err := teasersFrom(story.Teasers, headline)
	if err != nil {
		return Story{}, StoryKey{}, err
	}
	call, err := drawn("call", headlineOf(oneLine(story.Call)), callRunes)
	if err != nil {
		return Story{}, StoryKey{}, err
	}
	line, err := lineFrom(story.Line)
	if err != nil {
		return Story{}, StoryKey{}, err
	}
	hook, language := "", ""
	if experiment != "" {
		if hook, err = hookFrom(story.Hook); err != nil {
			return Story{}, StoryKey{}, err
		}
		if language, err = languageFrom(story.Language); err != nil {
			return Story{}, StoryKey{}, err
		}
		if hook == headline {
			return Story{}, StoryKey{}, fmt.Errorf("%w: the hook %q says the headline", errUndrawable, hook)
		}
	}
	if past.headlines.Contains(headline) {
		return Story{}, StoryKey{}, fmt.Errorf("%q was a headline of the last week, or of this run", headline)
	}
	if keys.Contains(key) {
		return Story{}, StoryKey{}, fmt.Errorf("%s in %s is told already in this run", key.Flag, key.Continent)
	}
	keys.Add(key)
	past.headlines.Add(headline)
	story.Headline, story.Teasers, story.Call, story.Line = headline, teasers, call, line
	story.Experiment, story.Hook, story.Language = experiment, hook, language
	return story, key, nil
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
		if telling.At.Before(now.Add(-reheadlining)) {
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

var errUndrawable = errors.New("the words cannot be drawn")

func headlineFrom(written string) (string, error) {
	return drawn("headline", headlineOf(oneLine(written)), headlineRunes)
}

// None is no line; one is drawn as written, not in capitals.
func lineFrom(written string) (string, error) {
	if line := oneLine(written); line != "" {
		return drawn("line", line, lineRunes)
	}
	return "", nil
}

func oneLine(written string) string {
	return strings.Join(strings.Fields(written), " ")
}

// Starts with a letter or a digit, so the generator never reads it as a flag.
func drawn(what, words string, runes int) (string, error) {
	if words == "" || utf8.RuneCountInString(words) > runes {
		return "", fmt.Errorf("%w: the %s %q is empty or over %d letters", errUndrawable, what, words, runes)
	}
	if first, _ := utf8.DecodeRuneInString(words); !unicode.IsLetter(first) && !unicode.IsDigit(first) {
		return "", fmt.Errorf("%w: the %s %q starts with %q", errUndrawable, what, words, first)
	}
	for _, r := range words {
		if !unicode.IsLetter(r) && !unicode.IsMark(r) && !unicode.IsDigit(r) && !strings.ContainsRune(" !?'’%:,.&-", r) {
			return "", fmt.Errorf("%w: the %s %q has %q", errUndrawable, what, words, r)
		}
	}
	return words, nil
}

// YouTube's title is a teaser and the brand.
var teaserRunes = youtubeTitleLength - utf8.RuneCountInString(brand)

var errTeasers = errors.New("the caption has no teaser")

func teasersFrom(written []string, headline string) ([]string, error) {
	var teasers []string
	for _, teaser := range written {
		said := headlineOf(oneLine(teaser))
		if said == "" || said == headline || utf8.RuneCountInString(said) > teaserRunes || slices.Contains(teasers, said) {
			continue
		}
		teasers = append(teasers, said)
	}
	if len(teasers) == 0 {
		return nil, fmt.Errorf("%w: none of %q is the headline in other words, in %d letters or fewer", errTeasers, written, teaserRunes)
	}
	return teasers, nil
}
