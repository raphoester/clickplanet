package bonuses

import (
	"errors"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules"
)

// Rules is what the rule set in force says of the boxes, the quizzes and the charges.
type Rules struct {
	Boxes Pace

	Kinds map[Kind]float64

	OfferTTL     time.Duration
	MissRetry    time.Duration
	ActiveWithin time.Duration

	Spread  SpreadRules
	Bomb    BombRules
	Enclose EncloseRules
	Shield  ShieldRules

	Welcome WelcomeRules

	Quiz QuizRules

	SignInRefill bool
}

// A tile keeps its shields in a byte.
const mostShieldsOnATile = 255

func (r Rules) Validate() error {
	return errors.Join(
		rules.Positive("boxes.minInterval", r.Boxes.MinInterval),
		rules.AtLeast("boxes.maxInterval", r.Boxes.MaxInterval, "boxes.minInterval", r.Boxes.MinInterval),
		rules.Positive("boxes.maxChargesPerHour", r.Boxes.MaxChargesPerHour),
		weightsProblem(r),
		rules.Positive("offerTtl", r.OfferTTL),
		rules.Positive("missRetry", r.MissRetry),
		rules.Positive("activeWithin", r.ActiveWithin),
		rules.Positive("spread.clicks", r.Spread.Clicks),
		rules.Positive("spread.maxPerBox", r.Spread.MaxPerBox),
		rules.Positive("bomb.rings", r.Bomb.Rings), rules.Finite("bomb.rings", r.Bomb.Rings),
		rules.Positive("enclose.maxTiles", r.Enclose.MaxTiles),
		rules.Positive("enclose.held", r.Enclose.Held),
		rules.Positive("enclose.maxPerBox", r.Enclose.MaxPerBox),
		rules.Positive("shield.held", r.Shield.Held),
		rules.Positive("shield.minPerBox", r.Shield.MinPerBox),
		rules.AtLeast("shield.maxPerBox", r.Shield.MaxPerBox, "shield.minPerBox", r.Shield.MinPerBox),
		rules.Positive("shield.perTile", r.Shield.PerTile),
		rules.AtMost("shield.perTile", r.Shield.PerTile, "what a tile keeps", mostShieldsOnATile),
		rules.NotNegative("welcome.within", r.Welcome.Within),
		rules.NotNegative("welcome.boxAfter", r.Welcome.BoxAfter),
		r.Quiz.validate(),
	)
}

// Each kind by its field in planet.v1.KindWeights.
var weighed = []struct {
	kind  Kind
	field string
}{
	{KindRefill, "refill"},
	{KindSpreadClicks, "spreadClicks"},
	{KindEncloseClicks, "encloseClicks"},
	{KindBomb, "bomb"},
	{KindShields, "shields"},
}

func weightsProblem(r Rules) error {
	var problems []error
	offered := false
	for _, each := range weighed {
		weight := r.Kinds[each.kind]
		problems = append(problems, rules.NotNegative("kinds."+each.field, weight), rules.Finite("kinds."+each.field, weight))
		offered = offered || r.Offered(each.kind)
	}
	if !offered {
		problems = append(problems, fmt.Errorf("kinds weigh every kind a box may be 0, so no box could be anything"))
	}

	return errors.Join(problems...)
}

func (q QuizRules) validate() error {
	return errors.Join(
		rules.Positive("quiz.minInterval", q.MinInterval),
		rules.AtLeast("quiz.maxInterval", q.MaxInterval, "quiz.minInterval", q.MinInterval),
		rules.Positive("quiz.maxChargesPerHour", q.MaxChargesPerHour),
		rules.Positive("quiz.missRetry", q.MissRetry),
		rules.Positive("quiz.offerTtl", q.OfferTTL),
		rules.Positive("quiz.answerWindow", q.AnswerWindow),
		rules.NotNegative("quiz.leaderBias", q.LeaderBias), rules.Finite("quiz.leaderBias", q.LeaderBias),
	)
}

type SpreadRules struct {
	Clicks    int
	MaxPerBox int
}

// BombRules: Off, no box or quiz offers a bomb, and a bomb held is not dropped.
type BombRules struct {
	Rings float64
	Off   bool
}

// Offered is true for a kind a box or a quiz may be: weighed above 0, and not a bomb while bombs are off.
func (r Rules) Offered(kind Kind) bool {
	return r.Kinds[kind] > 0 && (kind != KindBomb || !r.Bomb.Off)
}

type EncloseRules struct {
	MaxTiles  int
	Held      int
	MaxPerBox int
	Unlimited bool
}

type ShieldRules struct {
	Held      int
	MinPerBox int
	MaxPerBox int
	PerTile   int
}

type WelcomeRules struct {
	Within time.Duration

	BoxAfter time.Duration
}

type QuizRules struct {
	Enabled bool

	MinInterval time.Duration
	MaxInterval time.Duration

	MissRetry    time.Duration
	OfferTTL     time.Duration
	AnswerWindow time.Duration

	LeaderBias float64

	MaxChargesPerHour int
}
