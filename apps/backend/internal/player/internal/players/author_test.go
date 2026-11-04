package players_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
)

var authoredAt = time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC)

func TestAGuestAuthorIsItsCode(t *testing.T) {
	shown := players.GuestAuthor("91aa3d")

	assert.Equal(t, "guest_91aa3d", shown.Name())
	assert.True(t, shown.Guest())
	assert.False(t, shown.Admin())
	assert.Zero(t, shown.Color())
}

func TestANamedAuthorIsItsProfile(t *testing.T) {
	author := players.NamedAuthor(players.ProfileSnapshot{Account: players.AccountID{15: 1}, Name: "Ada", UpdatedAt: authoredAt, Admin: true, Color: 3}.Restore())

	assert.Equal(t, "Ada", author.Name())
	assert.False(t, author.Guest())
	assert.True(t, author.Admin())
	assert.Equal(t, players.Color(3), author.Color())
}

func TestARenamedAuthorKeepsItsMarkAndColorAndIsNoLongerAGuest(t *testing.T) {
	named := players.NamedAuthor(players.ProfileSnapshot{Account: players.AccountID{15: 1}, Name: "Ada", UpdatedAt: authoredAt, Admin: true, Color: 3}.Restore())

	renamed := named.Renamed("Ada_L")

	assert.Equal(t, "Ada_L", renamed.Name())
	assert.True(t, renamed.Admin())
	assert.Equal(t, players.Color(3), renamed.Color())
	assert.False(t, players.GuestAuthor("91aa3d").Renamed("Ada").Guest())
}
