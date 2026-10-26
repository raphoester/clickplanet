package rules_test

import (
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules"
)

func rushSnapshot() rules.Snapshot {
	rush := rules.TestSnapshot("rush")
	rush.Tempo = 3
	rush.Bonuses.Boxes = bonuses.Pace{MinInterval: time.Minute, MaxInterval: 2 * time.Minute, MaxChargesPerHour: 30}
	return rush
}

func TestASetIsWhatItWasMadeOf(t *testing.T) {
	set, err := rushSnapshot().Restore()
	require.NoError(t, err)

	assert.Equal(t, rushSnapshot(), set.TakeSnapshot())
}

func TestASetKeepsNothingItsSnapshotsShare(t *testing.T) {
	snapshot := rushSnapshot()
	set, err := snapshot.Restore()
	require.NoError(t, err)

	snapshot.Toll[0].Slowdown = 9
	snapshot.Bonuses.Kinds[bonuses.KindBomb] = 9
	taken := set.TakeSnapshot()
	taken.Toll[1].Share = 0.9

	assert.True(t, rushSnapshot().Equal(set.TakeSnapshot()))
}

func TestASetThatCannotPlayIsRefused(t *testing.T) {
	broken := map[string]func(*rules.Snapshot){
		"no name":                          func(s *rules.Snapshot) { s.Name = "" },
		"a name with capitals":             func(s *rules.Snapshot) { s.Name = "Rush" },
		"a name too long":                  func(s *rules.Snapshot) { s.Name = "a234567890123456789012345678901234" },
		"no tempo":                         func(s *rules.Snapshot) { s.Tempo = 0 },
		"a tempo that is no number":        func(s *rules.Snapshot) { s.Tempo = math.NaN() },
		"no refill":                        func(s *rules.Snapshot) { s.Clicks.PerSecond = 0 },
		"an endless refill":                func(s *rules.Snapshot) { s.Clicks.PerSecond = math.Inf(1) },
		"no bank":                          func(s *rules.Snapshot) { s.Clicks.Burst = 0 },
		"a scope smaller than an account":  func(s *rules.Snapshot) { s.Clicks.ScopeMultiplier = 0.5 },
		"a /64 smaller than an account":    func(s *rules.Snapshot) { s.Clicks.V6ScopeMultiplier = 0 },
		"guests smaller than an account":   func(s *rules.Snapshot) { s.Clicks.GuestScopeMultiplier = math.NaN() },
		"signing in that slows the refill": func(s *rules.Snapshot) { s.Clicks.LinkedMultiplier = 0.5 },
		"a new bank in debt":               func(s *rules.Snapshot) { s.Clicks.NewAccountClicks = -1 },
		"a new bank bigger than a bank":    func(s *rules.Snapshot) { s.Clicks.NewAccountClicks = 11 },
		"a toll slowdown of one": func(s *rules.Snapshot) {
			s.Toll = []clicks.TollStep{{Share: 0.1, Slowdown: 1}}
		},
		"a falling toll slowdown": func(s *rules.Snapshot) {
			s.Toll = []clicks.TollStep{{Share: 0.1, Slowdown: 1.5}, {Share: 0.2, Slowdown: 1.25}}
		},
		"a falling toll share": func(s *rules.Snapshot) {
			s.Toll = []clicks.TollStep{{Share: 0.5, Slowdown: 2}, {Share: 0.2, Slowdown: 3}}
		},
		"a toll share of zero":       func(s *rules.Snapshot) { s.Toll = []clicks.TollStep{{Share: 0, Slowdown: 2}} },
		"a toll share above the map": func(s *rules.Snapshot) { s.Toll = []clicks.TollStep{{Share: 1.5, Slowdown: 2}} },
		"an endless toll slowdown": func(s *rules.Snapshot) {
			s.Toll = []clicks.TollStep{{Share: 0.5, Slowdown: math.Inf(1)}}
		},
		"no shortest box wait":         func(s *rules.Snapshot) { s.Bonuses.Boxes.MinInterval = 0 },
		"a longest box wait too short": func(s *rules.Snapshot) { s.Bonuses.Boxes.MaxInterval = 30 * time.Second },
		"no hourly box cap":            func(s *rules.Snapshot) { s.Bonuses.Boxes.MaxChargesPerHour = 0 },
		"a kind that is not one":       func(s *rules.Snapshot) { s.Bonuses.Kinds["quadruple_clicks"] = 1 },
		"a kind not weighed":           func(s *rules.Snapshot) { delete(s.Bonuses.Kinds, bonuses.KindBomb) },
		"a weight below 0":             func(s *rules.Snapshot) { s.Bonuses.Kinds[bonuses.KindBomb] = -1 },
		"every weight 0": func(s *rules.Snapshot) {
			for kind := range s.Bonuses.Kinds {
				s.Bonuses.Kinds[kind] = 0
			}
		},
		"a box that never flies":         func(s *rules.Snapshot) { s.Bonuses.OfferTTL = 0 },
		"no miss retry":                  func(s *rules.Snapshot) { s.Bonuses.MissRetry = 0 },
		"nobody active":                  func(s *rules.Snapshot) { s.Bonuses.ActiveWithin = 0 },
		"no spread pool":                 func(s *rules.Snapshot) { s.Bonuses.Spread.Clicks = 0 },
		"a spread box of nothing":        func(s *rules.Snapshot) { s.Bonuses.Spread.MaxPerBox = 0 },
		"a bomb with no blast":           func(s *rules.Snapshot) { s.Bonuses.Bomb.Rings = 0 },
		"an enclosure of no tile":        func(s *rules.Snapshot) { s.Bonuses.Enclose.MaxTiles = 0 },
		"no enclosure stack":             func(s *rules.Snapshot) { s.Bonuses.Enclose.Held = 0 },
		"an enclose box of nothing":      func(s *rules.Snapshot) { s.Bonuses.Enclose.MaxPerBox = 0 },
		"no shield stack":                func(s *rules.Snapshot) { s.Bonuses.Shield.Held = 0 },
		"a shield box of nothing":        func(s *rules.Snapshot) { s.Bonuses.Shield.MinPerBox = 0 },
		"a shield box upside down":       func(s *rules.Snapshot) { s.Bonuses.Shield.MaxPerBox = 4 },
		"no shield on a tile":            func(s *rules.Snapshot) { s.Bonuses.Shield.PerTile = 0 },
		"more shields than a tile holds": func(s *rules.Snapshot) { s.Bonuses.Shield.PerTile = 256 },
		"a welcome in the past":          func(s *rules.Snapshot) { s.Bonuses.Welcome.Within = -time.Minute },
		"a first box before the account": func(s *rules.Snapshot) { s.Bonuses.Welcome.BoxAfter = -time.Minute },
		"no shortest quiz wait":          func(s *rules.Snapshot) { s.Bonuses.Quiz.MinInterval = 0 },
		"a longest quiz wait too short":  func(s *rules.Snapshot) { s.Bonuses.Quiz.MaxInterval = time.Minute },
		"no quiz miss retry":             func(s *rules.Snapshot) { s.Bonuses.Quiz.MissRetry = 0 },
		"a banner that never shows":      func(s *rules.Snapshot) { s.Bonuses.Quiz.OfferTTL = 0 },
		"no time to answer":              func(s *rules.Snapshot) { s.Bonuses.Quiz.AnswerWindow = 0 },
		"a lean below 0":                 func(s *rules.Snapshot) { s.Bonuses.Quiz.LeaderBias = -1 },
		"a lean that is no number":       func(s *rules.Snapshot) { s.Bonuses.Quiz.LeaderBias = math.NaN() },
		"no hourly quiz cap":             func(s *rules.Snapshot) { s.Bonuses.Quiz.MaxChargesPerHour = 0 },
	}
	for name, breaks := range broken {
		t.Run(name, func(t *testing.T) {
			snapshot := rushSnapshot()
			breaks(&snapshot)

			_, err := snapshot.Restore()
			require.ErrorIs(t, err, rules.ErrInvalidSet)
		})
	}
}

func TestEveryValueThatCanBeZeroIsASet(t *testing.T) {
	snapshot := rushSnapshot()
	snapshot.Toll = nil
	snapshot.Clicks.NewAccountClicks = 0
	snapshot.Bonuses.Kinds[bonuses.KindBomb] = 0
	snapshot.Bonuses.Welcome = bonuses.WelcomeRules{}
	snapshot.Bonuses.Quiz.LeaderBias = 0
	snapshot.Bonuses.Boxes.MaxInterval = snapshot.Bonuses.Boxes.MinInterval
	snapshot.Bonuses.Shield.PerTile = 255

	_, err := snapshot.Restore()
	require.NoError(t, err)
}

func TestARefusalNamesTheValue(t *testing.T) {
	snapshot := rushSnapshot()
	snapshot.Bonuses.Quiz.AnswerWindow = 0

	_, err := snapshot.Restore()
	require.ErrorContains(t, err, "quiz.answerWindow")
}
