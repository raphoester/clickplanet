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
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/titles"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/wearing"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

func TestTheAccountLeavesTheRosterAtOnce(t *testing.T) {
	now := time.Date(2026, 9, 17, 12, 0, 0, 0, time.UTC)
	visits := inmemory_visit_storage.New(cptime.NewFixedClock(now))
	visits.Record(presence.NewVisit(players.AccountID{15: 1}, wearing.AuthorOf(players.NamedAuthor(players.ProfileOf(players.AccountID{}, "Ada_L", time.Time{}, false, 0), players.Streak{}), titles.Standing{}), "aaaaaa", "fr", now))

	err := forget_visit_usecase.New(visits).Execute(t.Context(), players.AccountID{15: 1})

	require.NoError(t, err)
	assert.Empty(t, visits.Visits())
}
