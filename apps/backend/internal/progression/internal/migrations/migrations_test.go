package migrations_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	progressionv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/progression/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpoutbox"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg/cppgtest"
)

func TestTheLevelReachedHasAPlaceInTheOutbox(t *testing.T) {
	db := cppgtest.StartServer(t).OpenSchema(t, "progression", migrations.FS)

	require.NoError(t, db.InTx(t.Context(), func(ctx context.Context) error {
		return cpoutbox.New(db).Record(ctx, &progressionv1.LevelReached{AccountId: "00000000-0000-0000-0000-000000000001", Level: 2})
	}))
}
