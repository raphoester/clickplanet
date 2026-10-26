package rules

import (
	"fmt"
	"math"
	"regexp"
	"slices"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

var namePattern = regexp.MustCompile(`^[a-z][a-z0-9_-]{0,31}$`)

// A tile keeps its shields in a byte.
const mostShieldsOnATile = math.MaxUint8

func (s Snapshot) problem() error {
	if !namePattern.MatchString(string(s.Name)) {
		return fmt.Errorf("%w: a name is a lowercase letter, then up to 31 of a-z, 0-9, - and _", ErrInvalidSet)
	}

	for _, problem := range []error{
		atLeast("tempo", s.Tempo, math.SmallestNonzeroFloat64),
		throttleProblem(s.Clicks),
		tollProblem(s.Toll),
		paceProblem("boxes", s.Bonuses.Boxes.MinInterval, s.Bonuses.Boxes.MaxInterval, s.Bonuses.Boxes.MaxChargesPerHour),
		bonusesProblem(s.Bonuses),
		quizProblem(s.Bonuses.Quiz),
	} {
		if problem != nil {
			return fmt.Errorf("%w: %w", ErrInvalidSet, problem)
		}
	}

	return nil
}

func throttleProblem(throttle clicks.Throttle) error {
	return first(
		atLeast("clicks.perSecond", throttle.PerSecond, math.SmallestNonzeroFloat64),
		count("clicks.burst", throttle.Burst, 1),
		atLeast("clicks.scopeMultiplier", throttle.ScopeMultiplier, 1),
		atLeast("clicks.v6ScopeMultiplier", throttle.V6ScopeMultiplier, 1),
		atLeast("clicks.guestScopeMultiplier", throttle.GuestScopeMultiplier, 1),
		atLeast("clicks.linkedMultiplier", throttle.LinkedMultiplier, 1),
		atLeast("clicks.newAccountClicks", throttle.NewAccountClicks, 0),
		atMost("clicks.newAccountClicks", throttle.NewAccountClicks, float64(throttle.Burst)),
	)
}

func tollProblem(steps []clicks.TollStep) error {
	previous := clicks.TollStep{Slowdown: 1}

	for i, step := range steps {
		if math.IsNaN(step.Share) || step.Share <= previous.Share || step.Share > 1 {
			return fmt.Errorf("tollSteps[%d].share is %v: shares rise, above 0 and up to 1", i, step.Share)
		}
		if !finite(step.Slowdown) || step.Slowdown <= previous.Slowdown {
			return fmt.Errorf("tollSteps[%d].slowdown is %v: slowdowns rise, above 1", i, step.Slowdown)
		}
		previous = step
	}

	return nil
}

func paceProblem(of string, shortest, longest time.Duration, hourlyCap int) error {
	return first(
		longer(of+".minInterval", shortest, 0),
		asLong(of+".maxInterval", longest, shortest),
		count(of+".maxChargesPerHour", hourlyCap, 1),
	)
}

func bonusesProblem(rules bonuses.Rules) error {
	return first(
		kindsProblem(rules.Kinds),
		longer("bonuses.offerTtl", rules.OfferTTL, 0),
		longer("bonuses.missRetry", rules.MissRetry, 0),
		longer("bonuses.activeWithin", rules.ActiveWithin, 0),
		count("bonuses.spread.clicks", rules.Spread.Clicks, 1),
		count("bonuses.spread.maxPerBox", rules.Spread.MaxPerBox, 1),
		atLeast("bonuses.bomb.rings", rules.Bomb.Rings, math.SmallestNonzeroFloat64),
		count("bonuses.enclose.maxTiles", rules.Enclose.MaxTiles, 1),
		count("bonuses.enclose.held", rules.Enclose.Held, 1),
		count("bonuses.enclose.maxPerBox", rules.Enclose.MaxPerBox, 1),
		count("bonuses.shield.held", rules.Shield.Held, 1),
		count("bonuses.shield.minPerBox", rules.Shield.MinPerBox, 1),
		count("bonuses.shield.maxPerBox", rules.Shield.MaxPerBox, rules.Shield.MinPerBox),
		count("bonuses.shield.perTile", rules.Shield.PerTile, 1),
		atMost("bonuses.shield.perTile", float64(rules.Shield.PerTile), mostShieldsOnATile),
		asLong("bonuses.welcome.within", rules.Welcome.Within, 0),
		asLong("bonuses.welcome.boxAfter", rules.Welcome.BoxAfter, 0),
	)
}

func kindsProblem(kinds map[bonuses.Kind]float64) error {
	total := 0.0
	for kind, weight := range kinds {
		if !slices.Contains(bonuses.Kinds, kind) {
			return fmt.Errorf("bonuses.kinds holds %q, which is not one of %v", kind, bonuses.Kinds)
		}
		if !finite(weight) || weight < 0 {
			return fmt.Errorf("bonuses.kinds.%s is %v: a weight is a number of 0 or more", kind, weight)
		}
		total += weight
	}

	for _, kind := range bonuses.Kinds {
		if _, written := kinds[kind]; !written {
			return fmt.Errorf("bonuses.kinds has no %s: every kind is weighed", kind)
		}
	}
	if total == 0 {
		return fmt.Errorf("bonuses.kinds weighs every kind 0, so no box could be anything")
	}

	return nil
}

func quizProblem(quiz bonuses.QuizRules) error {
	return first(
		paceProblem("quiz", quiz.MinInterval, quiz.MaxInterval, quiz.MaxChargesPerHour),
		longer("quiz.missRetry", quiz.MissRetry, 0),
		longer("quiz.offerTtl", quiz.OfferTTL, 0),
		longer("quiz.answerWindow", quiz.AnswerWindow, 0),
		atLeast("quiz.leaderBias", quiz.LeaderBias, 0),
	)
}

func first(problems ...error) error {
	for _, problem := range problems {
		if problem != nil {
			return problem
		}
	}

	return nil
}

func finite(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}

func atLeast(name string, value, least float64) error {
	if !finite(value) || value < least {
		return fmt.Errorf("%s is %v, and must be a number of at least %v", name, value, least)
	}

	return nil
}

func atMost(name string, value, most float64) error {
	if value > most {
		return fmt.Errorf("%s is %v, and must be at most %v", name, value, most)
	}

	return nil
}

func count(name string, value, least int) error {
	if value < least {
		return fmt.Errorf("%s is %d, and must be at least %d", name, value, least)
	}

	return nil
}

func asLong(name string, value, least time.Duration) error {
	if value < least {
		return fmt.Errorf("%s is %s, and must be at least %s", name, value, least)
	}

	return nil
}

func longer(name string, value, than time.Duration) error {
	if value <= than {
		return fmt.Errorf("%s is %s, and must be longer than %s", name, value, than)
	}

	return nil
}
