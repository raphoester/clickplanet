package rules_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"

	boxesv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/boxes/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/boxes/internal/boxes"
	"github.com/raphoester/clickplanet.lol-backend/internal/boxes/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/boxes/internal/rules"
	"github.com/raphoester/clickplanet.lol-backend/internal/boxes/internal/rules/parts"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cprules"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

func TestThePostgresStoreKeepsTheContract(t *testing.T) {
	db := server.CreateDatabase(t).OpenSchema(t, "boxes", migrations.FS)

	suite.Run(t, &cprules.StoreContractSuite[*boxesv1.Rules]{
		NewStore: func() cprules.Store[*boxesv1.Rules] {
			require.NoError(t, db.Purge(t.Context()))
			return cprules.NewPostgresStore(db, parts.All())
		},
		Parts:   parts.All(),
		Default: rules.Seeded(),
		Rush: func() *boxesv1.Rules {
			return &boxesv1.Rules{MaxBoxesPerHour: proto.Uint32(30)}
		},
		Other: func() *boxesv1.Rules {
			return &boxesv1.Rules{MissRetry: durationpb.New(time.Minute)}
		},
	})
}

// With no bonuses schema to copy from, the migration seeds production's default.
func TestTheMigrationSeedsDefaultWithProductionsRules(t *testing.T) {
	db := server.CreateDatabase(t).OpenSchema(t, "boxes", migrations.FS)

	set, err := cprules.NewPostgresStore(db, parts.All()).Set(t.Context(), cprules.Default)
	require.NoError(t, err)

	assert.True(t, proto.Equal(rules.Seeded().Rules, set.TakeSnapshot().Rules), "%v", set.TakeSnapshot().Rules)
}

func TestAFrozenSetIsReadFrozen(t *testing.T) {
	db := server.CreateDatabase(t).OpenSchema(t, "boxes", migrations.FS)
	_, err := db.On(t.Context()).ExecContext(t.Context(), `INSERT INTO rule_sets (name, rules) VALUES ('frozen', '{"frozen": true}')`)
	require.NoError(t, err)

	set, err := cprules.NewPostgresStore(db, parts.All()).Set(t.Context(), "frozen")
	require.NoError(t, err)

	inForce := cprules.NewInForce[*boxesv1.Rules](cptime.SystemClock{})
	inForce.Enforce(set)
	assert.True(t, cprules.Reader(inForce, boxes.Part)().Frozen)
}
