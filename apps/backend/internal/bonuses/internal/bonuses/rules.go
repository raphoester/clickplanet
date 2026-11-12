package bonuses

import (
	"errors"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/quizzes"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/ratings"
)

// Rules is what the offers play (bonuses.v1.Rules): when a box or a quiz comes, what it may be and how much it grants.
// How much of each charge a hand holds, and what it does when it is spent, is planet's: an offer pays what it drew, and
// the wallet keeps what fits.
type Rules struct {
	Boxes Pace

	Kinds map[Kind]float64

	OfferTTL     time.Duration
	MissRetry    time.Duration
	ActiveWithin time.Duration

	Spread  SpreadRules
	Enclose EncloseRules
	Shield  ShieldRules

	Welcome WelcomeRules

	Quiz QuizRules

	SignInRefill bool

	// Frozen offers nothing, and refuses every claim and answer.
	Frozen bool
}

func (r Rules) Validate() error {
	return errors.Join(
		positive("boxes.minInterval", r.Boxes.MinInterval),
		atLeast("boxes.maxInterval", r.Boxes.MaxInterval, "boxes.minInterval", r.Boxes.MinInterval),
		positive("boxes.maxChargesPerHour", r.Boxes.MaxChargesPerHour),
		weightsProblem(r),
		positive("offerTtl", r.OfferTTL),
		positive("missRetry", r.MissRetry),
		positive("activeWithin", r.ActiveWithin),
		positive("spread.maxPerBox", r.Spread.MaxPerBox),
		positive("enclose.maxPerBox", r.Enclose.MaxPerBox),
		positive("shield.minPerBox", r.Shield.MinPerBox),
		atLeast("shield.maxPerBox", r.Shield.MaxPerBox, "shield.minPerBox", r.Shield.MinPerBox),
		notNegative("welcome.within", r.Welcome.Within),
		notNegative("welcome.boxAfter", r.Welcome.BoxAfter),
		r.Quiz.validate(),
	)
}

// Each kind by its field in bonuses.v1.KindWeights.
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
		problems = append(problems, notNegative("kinds."+each.field, weight), finite("kinds."+each.field, weight))
		offered = offered || r.Offered(each.kind)
	}
	if !offered {
		problems = append(problems, fmt.Errorf("kinds weigh every kind a box may be 0, so no box could be anything"))
	}

	return errors.Join(problems...)
}

func (q QuizRules) validate() error {
	return errors.Join(
		positive("quiz.minInterval", q.MinInterval),
		atLeast("quiz.maxInterval", q.MaxInterval, "quiz.minInterval", q.MinInterval),
		positive("quiz.maxChargesPerHour", q.MaxChargesPerHour),
		positive("quiz.missRetry", q.MissRetry),
		positive("quiz.offerTtl", q.OfferTTL),
		positive("quiz.answerWindow", q.AnswerWindow),
		notNegative("quiz.leaderBias", q.LeaderBias), finite("quiz.leaderBias", q.LeaderBias),
		chance("quiz.target", q.Fit.Target),
		notNegative("quiz.width", q.Fit.Width), finite("quiz.width", q.Fit.Width),
		share("quiz.floor", q.Fit.Floor), atMost("quiz.floor", q.Fit.Floor, "quiz.target", q.Fit.Target),
		positive("quiz.floorAfter", q.Fit.FloorAfter),
		share("quiz.explore", q.Fit.Explore),
	)
}

// chance is a chance of a right answer a rating can stand for.
func chance(name string, value float64) error {
	if value >= ratings.MinChance && value <= ratings.MaxChance {
		return nil
	}
	return fmt.Errorf("%s is %v, and a chance of a right answer is between %v and %v", name, value,
		ratings.MinChance, ratings.MaxChance)
}

type SpreadRules struct {
	MaxPerBox int
}

// Offered is true for a kind a box or a quiz may be: weighed above 0. A bomb while planet's bombs are off is a kind the
// wallet says no hand has room for.
func (r Rules) Offered(kind Kind) bool {
	return r.Kinds[kind] > 0
}

type EncloseRules struct {
	MaxPerBox int
}

type ShieldRules struct {
	MinPerBox int
	MaxPerBox int
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

	// Fit weighs the draw inside the subject the lean picked by how hard each question is; its Width 0 is off.
	Fit quizzes.Fit
}

// ErrFrozen refuses a claim, an opening and an answer while the offers play a frozen set or planet's map takes no play.
var ErrFrozen = errors.New("no box or quiz can be taken now: the game is frozen")
