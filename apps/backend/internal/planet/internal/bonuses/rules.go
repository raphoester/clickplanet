package bonuses

import "time"

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
}

type InForce interface {
	Bonuses() Rules
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
