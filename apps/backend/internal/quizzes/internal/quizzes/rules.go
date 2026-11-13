package quizzes

import (
	"errors"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/quizzes/internal/bank"
	"github.com/raphoester/clickplanet.lol-backend/internal/quizzes/internal/ratings"
)

// Rules is what the quizzes play (quizzes.v1.Rules): when a quiz comes, how long it stays and how its question is drawn.
// What a right answer pays is the bonuses module's.
type Rules struct {
	Enabled bool

	MinInterval time.Duration
	MaxInterval time.Duration

	MissRetry    time.Duration
	OfferTTL     time.Duration
	AnswerWindow time.Duration

	LeaderBias float64

	MaxRightAnswersPerHour int

	// Fit weighs the draw inside the subject the lean picked by how hard each question is; its Width 0 is off.
	Fit bank.Fit

	ActiveWithin time.Duration

	Welcome WelcomeRules

	// Frozen offers nothing, and refuses every opening and answer.
	Frozen bool
}

type WelcomeRules struct {
	Within time.Duration
}

// pace is what a caller's next quiz is drawn by: a change of it moves the quizzes already due.
type pace struct {
	enabled     bool
	minInterval time.Duration
	maxInterval time.Duration
}

func (r Rules) pace() pace {
	return pace{enabled: r.Enabled, minInterval: r.MinInterval, maxInterval: r.MaxInterval}
}

func (r Rules) Validate() error {
	return errors.Join(
		positive("minInterval", r.MinInterval),
		atLeast("maxInterval", r.MaxInterval, "minInterval", r.MinInterval),
		positive("maxRightAnswersPerHour", r.MaxRightAnswersPerHour),
		positive("missRetry", r.MissRetry),
		positive("offerTtl", r.OfferTTL),
		positive("answerWindow", r.AnswerWindow),
		notNegative("leaderBias", r.LeaderBias), finite("leaderBias", r.LeaderBias),
		chance("target", r.Fit.Target),
		notNegative("width", r.Fit.Width), finite("width", r.Fit.Width),
		share("floor", r.Fit.Floor), atMost("floor", r.Fit.Floor, "target", r.Fit.Target),
		positive("floorAfter", r.Fit.FloorAfter),
		share("explore", r.Fit.Explore),
		positive("activeWithin", r.ActiveWithin),
		notNegative("welcome.within", r.Welcome.Within),
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

// ErrFrozen refuses an opening and an answer while the quizzes play a frozen set or planet's map takes no play.
var ErrFrozen = errors.New("no quiz can be taken now: the game is frozen")
