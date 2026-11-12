package migrations_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	missionsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/missions/v1"
	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	scoringv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/scoring/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpoutbox"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func TestTheOutboxKeepsEveryMessageMissionsSends(t *testing.T) {
	db := cppg.StartTestServer(t).OpenSchema(t, "missions", migrations.FS)

	require.NoError(t, db.InTx(t.Context(), func(ctx context.Context) error {
		return cpoutbox.New(db).Record(ctx, &planetv1.GrantCharges{}, &scoringv1.GrantPoints{}, &missionsv1.MissionsPaid{},
			&missionsv1.FlagMissionDone{})
	}))
}
