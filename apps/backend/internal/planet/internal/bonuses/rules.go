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

	AdRefill AdRefillRules
}

// A tile keeps its shields in a byte.
const mostShieldsOnATile = 255

func (r Rules) Validate() error {
	return errors.Join(
		rules.Positive("boxes.minInterval", r.Boxes.MinInterval),
		rules.AtLeast("boxes.maxInterval", r.Boxes.MaxInterval, "boxes.minInterval", r.Boxes.MinInterval),
		rules.Positive("boxes.maxChargesPerHour", r.Boxes.MaxChargesPerHour),
		weightsProblem(r.Kinds),
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
		rules.Positive("adRefill.every", r.AdRefill.Every),
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

func weightsProblem(weights map[Kind]float64) error {
	var problems []error
	total := 0.0
	for _, each := range weighed {
		weight := weights[each.kind]
		problems = append(problems, rules.NotNegative("kinds."+each.field, weight), rules.Finite("kinds."+each.field, weight))
		total += weight
	}
	if total <= 0 {
		problems = append(problems, fmt.Errorf("kinds weigh every kind 0, so no box could be anything"))
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

type BombRules struct {
	Rings float64
}

type EncloseRules struct {
	MaxTiles  int
	Held      int
	MaxPerBox int
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

type AdRefillRules struct {
	Enabled bool

	Every time.Duration
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
