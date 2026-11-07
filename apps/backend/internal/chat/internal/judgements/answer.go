package judgements

import (
	"strings"
	"unicode/utf8"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
)

// What the model may answer of a report, and nothing else: the schema of its structured output lists these.
type Category string

const (
	// Nothing the rules take out: the report is dismissed.
	CategoryNone Category = "none"
	// What goes: a slur, a Nazi symbol or slogan, a threat, sexual content, personal information, hateful politics, a
	// shock GIF.
	CategoryOffence Category = "offence"
	CategorySpam    Category = "spam"
	// Never acted on: the operator's.
	CategoryChildSafety Category = "child_safety"
	CategorySelfHarm    Category = "self_harm"
	CategoryCheating    Category = "cheating"
	CategoryUnsure      Category = "unsure"
)

var Categories = []Category{
	CategoryNone, CategoryOffence, CategorySpam, CategoryChildSafety, CategorySelfHarm, CategoryCheating, CategoryUnsure,
}

// The model's judgement of a report. What it leads to is Decide's, by the policy's steps.
type Answer struct {
	Category Category
	// A slur, a Nazi symbol or slogan, a threat, sexual content aimed at a player, personal information, porn or gore.
	Severe bool
	// The reported player's messages that break the rules: only those may be removed.
	Offending []messages.MessageID
	// A few words of what was said, never a quote of it: the reason every action carries.
	What string
	// One line of why, for the decision and the log.
	Why   string
	Spend Spend
}

// What a call cost, in tokens: what the log says of each judgement.
type Spend struct {
	Model      string
	Input      int64
	CacheWrite int64
	CacheRead  int64
	Output     int64
}

// Claude Haiku 5.5's price, in dollars per million tokens, while a prompt stays at 100K or fewer: a judgement is
// a few thousand.
const (
	inputPrice  = 0.10
	outputPrice = 0.50
)

func (s Spend) Dollars() float64 {
	return (float64(s.Input)*inputPrice + float64(s.CacheWrite)*inputPrice*1.25 + float64(s.CacheRead)*inputPrice*0.1 +
		float64(s.Output)*outputPrice) / 1e6
}

const maxWhat = 120

// One line, short, and never empty: the model writes it, and it goes into every reason.
func whatOf(answered string) string {
	what := strings.Join(strings.Fields(answered), " ")
	if utf8.RuneCountInString(what) > maxWhat {
		what = string([]rune(what)[:maxWhat])
	}
	if what == "" {
		return "a message against the rules"
	}
	return what
}

func whyOf(answered string) string {
	why := strings.Join(strings.Fields(answered), " ")
	if utf8.RuneCountInString(why) > maxWhy {
		why = string([]rune(why)[:maxWhy])
	}
	return why
}

const maxWhy = 300
