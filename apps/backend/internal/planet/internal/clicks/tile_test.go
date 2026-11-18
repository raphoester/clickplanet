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

func TestAClaimForTheFlagTheTileWearsChangesNothing(t *testing.T) {
	tile := clicks.TileSnapshot{ID: 7, Owner: "fr", Shields: 3, Holder: ana}.Restore()

	claimed, impact := tile.Claimed("fr", bob)

	assert.Equal(t, tile, claimed)
	assert.Equal(t, clicks.Impact{Tile: 7, Owner: "fr", Outcome: clicks.Unchanged, Shields: 3}, impact)
}
