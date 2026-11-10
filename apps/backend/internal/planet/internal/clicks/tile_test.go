package clicks_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
)

var (
	ana = cpsession.AccountID(uuid.MustParse("0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11"))
	bob = cpsession.AccountID(uuid.MustParse("6f1c2a3e-8b4d-4e5f-9a6b-7c8d9e0f1a2b"))
)

func TestATakeSendsTheShieldsHomeAndHoldsTheTileForItsTaker(t *testing.T) {
	tile := clicks.TileSnapshot{ID: 7, Owner: "de", Holder: ana}.Restore()

	claimed, impact := tile.Claimed("fr", bob)

	assert.Equal(t, clicks.TileSnapshot{ID: 7, Owner: "fr", Holder: bob}, claimed.TakeSnapshot())
	assert.Equal(t, clicks.Impact{Tile: 7, Owner: "de", Outcome: clicks.Taken}, impact)
}

func TestAClaimOnAShieldStrikesOneAndKeepsTheFlagAndItsHolder(t *testing.T) {
	tile := clicks.TileSnapshot{ID: 7, Owner: "de", Shields: 3, Holder: ana}.Restore()

	claimed, impact := tile.Claimed("fr", bob)

	assert.Equal(t, clicks.TileSnapshot{ID: 7, Owner: "de", Shields: 2, Holder: ana}, claimed.TakeSnapshot())
	assert.Equal(t, clicks.Impact{Tile: 7, Owner: "de", Outcome: clicks.Shielded, Shields: 2}, impact)
}

func TestAClaimForTheFlagTheTileWearsChangesNothing(t *testing.T) {
	tile := clicks.TileSnapshot{ID: 7, Owner: "fr", Shields: 3, Holder: ana}.Restore()

	claimed, impact := tile.Claimed("fr", bob)

	assert.Equal(t, tile, claimed)
	assert.Equal(t, clicks.Impact{Tile: 7, Owner: "fr", Outcome: clicks.Unchanged, Shields: 3}, impact)
}

func TestAnEmptyTileIsTakenAndHeldByNobodyForAClaimWithNoAccount(t *testing.T) {
	claimed, impact := clicks.TileSnapshot{ID: 7}.Restore().Claimed("fr", cpsession.NoAccount)

	assert.Equal(t, clicks.TileSnapshot{ID: 7, Owner: "fr"}, claimed.TakeSnapshot())
	assert.Equal(t, clicks.Taken, impact.Outcome)
}
