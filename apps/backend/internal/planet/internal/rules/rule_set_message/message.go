package rule_set_message

import (
	"fmt"
	"time"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/durationpb"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules"
)

func Encode(snapshot rules.Snapshot) *planetv1.RuleSet {
	throttle, game := snapshot.Clicks, snapshot.Bonuses

	return &planetv1.RuleSet{
		Name:  string(snapshot.Name),
		Tempo: snapshot.Tempo,
		Boxes: &planetv1.BoxPace{
			MinInterval:       durationpb.New(game.Boxes.MinInterval),
			MaxInterval:       durationpb.New(game.Boxes.MaxInterval),
			MaxChargesPerHour: count(game.Boxes.MaxChargesPerHour),
		},
		Clicks: &planetv1.ClickRules{
			PerSecond:            throttle.PerSecond,
			Burst:                count(throttle.Burst),
			ScopeMultiplier:      throttle.ScopeMultiplier,
			V6ScopeMultiplier:    throttle.V6ScopeMultiplier,
			GuestScopeMultiplier: throttle.GuestScopeMultiplier,
			LinkedMultiplier:     throttle.LinkedMultiplier,
			NewAccountClicks:     throttle.NewAccountClicks,
		},
		TollSteps: tollStepsOf(snapshot.Toll),
		Bonuses: &planetv1.BonusRules{
			Kinds: &planetv1.KindWeights{
				Refill:        game.Kinds[bonuses.KindRefill],
				SpreadClicks:  game.Kinds[bonuses.KindSpreadClicks],
				EncloseClicks: game.Kinds[bonuses.KindEncloseClicks],
				Bomb:          game.Kinds[bonuses.KindBomb],
				Shields:       game.Kinds[bonuses.KindShields],
			},
			OfferTtl:     durationpb.New(game.OfferTTL),
			MissRetry:    durationpb.New(game.MissRetry),
			ActiveWithin: durationpb.New(game.ActiveWithin),
			Spread:       &planetv1.SpreadRules{Clicks: count(game.Spread.Clicks), MaxPerBox: count(game.Spread.MaxPerBox)},
			Bomb:         &planetv1.BombRules{Rings: game.Bomb.Rings},
			Enclose: &planetv1.EncloseRules{
				MaxTiles: count(game.Enclose.MaxTiles), Held: count(game.Enclose.Held), MaxPerBox: count(game.Enclose.MaxPerBox),
			},
			Shield: &planetv1.ShieldRules{
				Held: count(game.Shield.Held), MinPerBox: count(game.Shield.MinPerBox),
				MaxPerBox: count(game.Shield.MaxPerBox), PerTile: count(game.Shield.PerTile),
			},
			Welcome: &planetv1.WelcomeRules{
				Within: durationpb.New(game.Welcome.Within), BoxAfter: durationpb.New(game.Welcome.BoxAfter),
			},
		},
		Quiz: &planetv1.QuizRules{
			Enabled:           game.Quiz.Enabled,
			MinInterval:       durationpb.New(game.Quiz.MinInterval),
			MaxInterval:       durationpb.New(game.Quiz.MaxInterval),
			MissRetry:         durationpb.New(game.Quiz.MissRetry),
			OfferTtl:          durationpb.New(game.Quiz.OfferTTL),
			AnswerWindow:      durationpb.New(game.Quiz.AnswerWindow),
			LeaderBias:        game.Quiz.LeaderBias,
			MaxChargesPerHour: count(game.Quiz.MaxChargesPerHour),
		},
	}
}

func Decode(set *planetv1.RuleSet) rules.Snapshot {
	throttle, game, quiz := set.GetClicks(), set.GetBonuses(), set.GetQuiz()
	kinds := game.GetKinds()

	return rules.Snapshot{
		Name:  rules.SetName(set.GetName()),
		Tempo: set.GetTempo(),
		Clicks: clicks.Throttle{
			PerSecond:            throttle.GetPerSecond(),
			Burst:                int(throttle.GetBurst()),
			ScopeMultiplier:      throttle.GetScopeMultiplier(),
			V6ScopeMultiplier:    throttle.GetV6ScopeMultiplier(),
			GuestScopeMultiplier: throttle.GetGuestScopeMultiplier(),
			LinkedMultiplier:     throttle.GetLinkedMultiplier(),
			NewAccountClicks:     throttle.GetNewAccountClicks(),
		},
		Toll: tollOf(set.GetTollSteps()),
		Bonuses: bonuses.Rules{
			Boxes: bonuses.Pace{
				MinInterval:       duration(set.GetBoxes().GetMinInterval()),
				MaxInterval:       duration(set.GetBoxes().GetMaxInterval()),
				MaxChargesPerHour: int(set.GetBoxes().GetMaxChargesPerHour()),
			},
			Kinds: map[bonuses.Kind]float64{
				bonuses.KindRefill:        kinds.GetRefill(),
				bonuses.KindSpreadClicks:  kinds.GetSpreadClicks(),
				bonuses.KindEncloseClicks: kinds.GetEncloseClicks(),
				bonuses.KindBomb:          kinds.GetBomb(),
				bonuses.KindShields:       kinds.GetShields(),
			},
			OfferTTL:     duration(game.GetOfferTtl()),
			MissRetry:    duration(game.GetMissRetry()),
			ActiveWithin: duration(game.GetActiveWithin()),
			Spread:       bonuses.SpreadRules{Clicks: int(game.GetSpread().GetClicks()), MaxPerBox: int(game.GetSpread().GetMaxPerBox())},
			Bomb:         bonuses.BombRules{Rings: game.GetBomb().GetRings()},
			Enclose: bonuses.EncloseRules{
				MaxTiles:  int(game.GetEnclose().GetMaxTiles()),
				Held:      int(game.GetEnclose().GetHeld()),
				MaxPerBox: int(game.GetEnclose().GetMaxPerBox()),
			},
			Shield: bonuses.ShieldRules{
				Held:      int(game.GetShield().GetHeld()),
				MinPerBox: int(game.GetShield().GetMinPerBox()),
				MaxPerBox: int(game.GetShield().GetMaxPerBox()),
				PerTile:   int(game.GetShield().GetPerTile()),
			},
			Welcome: bonuses.WelcomeRules{
				Within: duration(game.GetWelcome().GetWithin()), BoxAfter: duration(game.GetWelcome().GetBoxAfter()),
			},
			Quiz: bonuses.QuizRules{
				Enabled:           quiz.GetEnabled(),
				MinInterval:       duration(quiz.GetMinInterval()),
				MaxInterval:       duration(quiz.GetMaxInterval()),
				MissRetry:         duration(quiz.GetMissRetry()),
				OfferTTL:          duration(quiz.GetOfferTtl()),
				AnswerWindow:      duration(quiz.GetAnswerWindow()),
				LeaderBias:        quiz.GetLeaderBias(),
				MaxChargesPerHour: int(quiz.GetMaxChargesPerHour()),
			},
		},
	}
}

// Document is how postgres keeps a set: its message's JSON, with no name, which is the row's.
func Document(snapshot rules.Snapshot) ([]byte, error) {
	set := Encode(snapshot)
	set.Name = ""

	document, err := protojson.Marshal(set)
	if err != nil {
		return nil, fmt.Errorf("failed to write the rule set %q: %w", snapshot.Name, err)
	}
	return document, nil
}

func Parse(name string, document []byte) (*planetv1.RuleSet, error) {
	set := &planetv1.RuleSet{}
	if err := protojson.Unmarshal(document, set); err != nil {
		return nil, fmt.Errorf("failed to read the rule set %q: %w", name, err)
	}
	set.Name = name

	return set, nil
}

func tollStepsOf(steps []clicks.TollStep) []*planetv1.TollStep {
	out := make([]*planetv1.TollStep, len(steps))
	for i, step := range steps {
		out[i] = &planetv1.TollStep{Share: step.Share, Slowdown: step.Slowdown}
	}

	return out
}

func tollOf(steps []*planetv1.TollStep) []clicks.TollStep {
	if len(steps) == 0 {
		return nil
	}

	out := make([]clicks.TollStep, len(steps))
	for i, step := range steps {
		out[i] = clicks.TollStep{Share: step.GetShare(), Slowdown: step.GetSlowdown()}
	}

	return out
}

func duration(d *durationpb.Duration) time.Duration {
	if d == nil {
		return 0
	}

	return d.AsDuration()
}

func count(n int) uint32 {
	return uint32(max(n, 0))
}
