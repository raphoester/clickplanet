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

// A headline is drawn over the globe in a font of a few sizes, two lines at most for this many letters.
const headlineRunes = 60

// What the editor is offered: never a story the rules would not tell.
type Offer struct {
	// How many clips the run makes.
	Count int
	// Told first whatever the editor says, with their own headlines: a day won, the day's news.
	First Shortlist
	// The editor's to rank and headline: not told in the last 3 days, no headline of the last week, and their flag in
	// fewer than 2 of the last posts.
	Stories Shortlist
	// The posts of the last 3 days, newest first, for the editor to vary on.
	Recent []Telling
	// How many of the last posts showed each flag: a flag in 2 waits, so a flag in 1 takes one clip of the run at most.
	Shown map[string]int
	Now   time.Time
}

// How many stories of the run are the editor's.
func (o Offer) Slots() int {
	return max(o.Count-len(o.First), 0)
}

// Whether the editor has anything to decide: a quiet run, or a run of days won alone, asks nobody.
func (o Offer) Open() bool {
	return o.Slots() > 0 && len(o.Stories) > 0
}

// Recent are the posts of the last retelling window, Shown the flags of the last `lately` of them.
func (s Shortlist) Offer(count int, told []Telling, now time.Time) Offer {
	past := pastOf(told, now)
	offer := Offer{Count: count, Shown: map[string]int{}, Now: now}
	for _, story := range s {
		switch {
		case !past.untold(story):
		case story.Key.Angle == DayWon:
			if len(offer.First) < count {
				offer.First = append(offer.First, story)
			}
		case past.crowded(story.Key.Flag):
		default:
			offer.Stories = append(offer.Stories, story)
		}
	}
	for _, flags := range past.shown {
		flags.ForEach(func(flag string) { offer.Shown[flag]++ })
	}
	for _, telling := range told {
		if !telling.At.Before(now.Add(-retelling)) {
			offer.Recent = append(offer.Recent, telling)
		}
	}
	slices.SortStableFunc(offer.Recent, func(a, b Telling) int { return b.At.Compare(a.At) })
	return offer
}

// A story the editor tells, and its headline.
type Edit struct {
	Pick     Pick
	Headline string
}

// The editor's answer: the stories of the offer it would tell, best first, each with its headline, and why.
type Edition struct {
	Edits []Edit
	Why   string
}

// What one call to the model cost. Zero when nothing was sent.
type Spend struct {
	Model            string
	InputTokens      int64
	CacheWriteTokens int64
	CacheReadTokens  int64
	OutputTokens     int64
	// Estimated from the model's prices: the bill is the provider's.
	USD float64
}

// Who chose the run, as its summary and its log say it: the model, or the rules when it could not.
type Editorial struct {
	// Whether the model was called: not on a run with nothing to decide.
	Asked bool
	// Why the rules made the run (no key, no answer, a refusal, one that did not hold); empty when the model did.
	FellBack string
	// The model's own reason.
	Why   string
	Spend Spend
}

var ErrInvalidEdition = errors.New("the edition does not hold")

// The run as the editor would tell it, within the rules: the days won first, then the editor's order, each story
// still skipped when an earlier one of the run filled its flag's share, and the stories it left out last, in the
// rules' order, so the run makes as many clips as the rules would. Also each headline the editor changed, by pick.
func (s Shortlist) Edited(offer Offer, edition Edition, told []Telling, now time.Time) (Choice, map[Pick]string, error) {
	headlines, err := s.checked(offer, edition, pastOf(told, now))
	if err != nil {
		return Choice{}, nil, err
	}

	retitled := slices.Clone(s)
	order := make([]Pick, 0, len(edition.Edits))
	for _, edit := range edition.Edits {
		order = append(order, edit.Pick)
		if headline, changed := headlines[edit.Pick]; changed {
			retitled[edit.Pick-1].Title = headline
		}
	}
	next := func(past *past, _ *cpcolls.Set[Angle]) (Story, bool) {
		if story, found := retitled.dayWon(past); found {
			return story, true
		}
		for _, pick := range order {
			if story := retitled[pick-1]; past.untold(story) && !past.crowded(story.Key.Flag) {
				return story, true
			}
		}
		return retitled.freshest(past, func(Story) bool { return true })
	}
	return retitled.walk(offer.Count, pastOf(told, now), next), headlines, nil
}

// Each headline that differs from the generator's, upper case, by pick; an error for an edition that strays from
// the offer or a headline the rules would skip.
func (s Shortlist) checked(offer Offer, edition Edition, past *past) (map[Pick]string, error) {
	if len(edition.Edits) == 0 {
		return nil, fmt.Errorf("%w: it tells no story", ErrInvalidEdition)
	}
	offered := cpcolls.NewSet(offer.Stories.Picks()...)
	taken := cpcolls.NewSet[string]()
	for _, story := range offer.First {
		taken.Add(headlineOf(story.Title))
	}

	edited := cpcolls.NewSet[Pick]()
	headlines := map[Pick]string{}
	for _, edit := range edition.Edits {
		if !offered.Contains(edit.Pick) || edited.Contains(edit.Pick) {
			return nil, fmt.Errorf("%w: story %d is not offered, or told twice", ErrInvalidEdition, edit.Pick)
		}
		edited.Add(edit.Pick)

		story := s[edit.Pick-1]
		headline, err := headlineFrom(edit.Headline)
		if err != nil {
			return nil, fmt.Errorf("%w: story %d: %w", ErrInvalidEdition, edit.Pick, err)
		}
		if headline != headlineOf(story.Title) && past.headlines.Contains(headline) {
			return nil, fmt.Errorf("%w: story %d: %q was a headline of the last week", ErrInvalidEdition, edit.Pick, headline)
		}
		if taken.Contains(headline) {
			return nil, fmt.Errorf("%w: story %d: %q is another story's headline", ErrInvalidEdition, edit.Pick, headline)
		}
		if slices.ContainsFunc(story.Teasers, func(teaser string) bool { return headlineOf(teaser) == headline }) {
			return nil, fmt.Errorf("%w: story %d: %q is a teaser of its caption", ErrInvalidEdition, edit.Pick, headline)
		}
		taken.Add(headline)
		if headline != story.Title {
			headlines[edit.Pick] = headline
		}
	}
	return headlines, nil
}

var errHeadline = errors.New("the headline cannot be drawn")

// Upper case, its spaces one each: letters of any script, digits, and the punctuation of a headline, as the clip's
// font draws them, starting with a letter or a digit, so the generator never reads it as a flag. No emoji, no line
// break, no quotes to close.
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
