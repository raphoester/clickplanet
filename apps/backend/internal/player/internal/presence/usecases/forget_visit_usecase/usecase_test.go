package forget_visit_usecase_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/presence"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/presence/inmemory_visit_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/presence/usecases/forget_visit_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

func TestTheAccountLeavesTheRosterAtOnce(t *testing.T) {
	now := time.Date(2026, 9, 17, 12, 0, 0, 0, time.UTC)
	visits := inmemory_visit_storage.New(cptime.NewFixedClock(now))
	visits.Record(presence.Visit{Account: players.AccountID{15: 1}, Author: players.Author{Name: "Ada_L"}, Tag: "aaaaaa", Country: "fr", At: now})

	err := forget_visit_usecase.New(visits).Execute(t.Context(), players.AccountID{15: 1})

	require.NoError(t, err)
	assert.Empty(t, visits.Visits())
}
