package players_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
)

var profiledAt = time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC)

func TestARenamedPlayerKeepsItsMarkAndColorAndIsNoLongerAGuest(t *testing.T) {
	named := players.NamedPlayer(players.ProfileSnapshot{Account: players.AccountID{15: 1}, Name: "Ada", UpdatedAt: profiledAt, Admin: true, Color: 3}.Restore())

	renamed := named.Renamed("Ada_L")

	assert.Equal(t, "Ada_L", renamed.Name())
	assert.True(t, renamed.Admin())
	assert.Equal(t, players.Color(3), renamed.Color())
	assert.False(t, players.GuestPlayer("91aa3d").Renamed("Ada").Guest())
}
