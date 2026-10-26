package postgres_rule_set_store_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules/postgres_rule_set_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func TestThePostgresStoreKeepsTheContract(t *testing.T) {
	db := cppg.StartTestServer(t).OpenSchema(t, "planet", migrations.FS)

	suite.Run(t, &rules.StoreContractSuite{
		NewStore: func() rules.Store {
			require.NoError(t, db.Purge(t.Context()))
			return postgres_rule_set_store.New(db)
		},
	})
}

func TestTheMigrationKeepsTheGamesOwnRulesAsTheDefault(t *testing.T) {
	db := cppg.StartTestServer(t).OpenSchema(t, "planet", migrations.FS)

	set, err := postgres_rule_set_store.New(db).Set(t.Context(), rules.Default)
	require.NoError(t, err)

	assert.Equal(t, rules.Snapshot{
		Name:  rules.Default,
		Tempo: 1,
		Clicks: clicks.Throttle{
			PerSecond: 0.2, Burst: 60,
			ScopeMultiplier: 10, V6ScopeMultiplier: 4, GuestScopeMultiplier: 1, LinkedMultiplier: 2,
			NewAccountClicks: 10,
		},
		Toll: []clicks.TollStep{{Share: 0.1, Slowdown: 1.5}, {Share: 0.2, Slowdown: 2.5}, {Share: 0.3, Slowdown: 4}},
		Bonuses: bonuses.Rules{
			Boxes: bonuses.Pace{MinInterval: 4 * time.Minute, MaxInterval: 8 * time.Minute, MaxChargesPerHour: 12},
			Kinds: map[bonuses.Kind]float64{
				bonuses.KindRefill: 5, bonuses.KindSpreadClicks: 3, bonuses.KindEncloseClicks: 2, bonuses.KindBomb: 1,
				bonuses.KindShields: 3,
			},
			OfferTTL:     15 * time.Second,
			MissRetry:    2 * time.Minute,
			ActiveWithin: 2 * time.Minute,
			Spread:       bonuses.SpreadRules{Clicks: 8, MaxPerBox: 4},
			Bomb:         bonuses.BombRules{Rings: 4},
			Enclose:      bonuses.EncloseRules{MaxTiles: 25, Held: 3, MaxPerBox: 3},
			Shield:       bonuses.ShieldRules{Held: 30, MinPerBox: 5, MaxPerBox: 20, PerTile: 10},
			Welcome:      bonuses.WelcomeRules{Within: 10 * time.Minute, BoxAfter: time.Minute},
			Quiz: bonuses.QuizRules{
				Enabled: true, MinInterval: 4 * time.Minute, MaxInterval: 8 * time.Minute, MissRetry: 2 * time.Minute,
				OfferTTL: 25 * time.Second, AnswerWindow: 8 * time.Second, LeaderBias: 1, MaxChargesPerHour: 12,
			},
		},
	}, set.TakeSnapshot(), "production's values on 26 October 2026")
}

func TestASetKeptThatCannotPlayIsRefused(t *testing.T) {
	db := cppg.StartTestServer(t).OpenSchema(t, "planet", migrations.FS)
	_, err := db.On(t.Context()).ExecContext(t.Context(),
		`UPDATE rule_sets SET rules = jsonb_set(rules, '{clicks,burst}', '0') WHERE name = 'default'`)
	require.NoError(t, err)

	_, err = postgres_rule_set_store.New(db).Set(t.Context(), rules.Default)

	require.ErrorIs(t, err, rules.ErrInvalidSet)
}

func TestASetKeptWithAValueThisBuildDoesNotKnowIsRefused(t *testing.T) {
	db := cppg.StartTestServer(t).OpenSchema(t, "planet", migrations.FS)
	_, err := db.On(t.Context()).ExecContext(t.Context(),
		`UPDATE rule_sets SET rules = jsonb_set(rules, '{frozen}', 'true') WHERE name = 'default'`)
	require.NoError(t, err)

	_, err = postgres_rule_set_store.New(db).Set(t.Context(), rules.Default)

	require.ErrorContains(t, err, "frozen")
}
