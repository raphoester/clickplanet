package players_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
)

func TestEachCareerIsItsStatsAndWhenItsAccountWasMade(t *testing.T) {
	made := time.Date(2026, 9, 20, 0, 0, 0, 0, time.UTC)
	page := []players.Stats{{Account: players.AccountID{15: 1}, TilesTaken: 3}, {Account: players.AccountID{15: 2}}}

	careers := players.CareersOf(page, map[players.AccountID]time.Time{{15: 1}: made})

	assert.Equal(t, []players.Career{{Stats: page[0], CreatedAt: made}, {Stats: page[1]}}, careers)
	assert.Equal(t, []players.AccountID{{15: 1}, {15: 2}}, players.AccountsOf(page))
}
