package planetmessage_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/planetmessage"
)

func TestAWholeMapIsTwoBytesATileFromTheFirstHeldIntoTheFlagsSortedAfterNoOne(t *testing.T) {
	whole := planetmessage.Whole(map[uint32]ledger.Owner{
		3: {Country: "fr"}, 4: {}, 6: {Country: "de", At: time.Unix(60, 0)},
	})

	assert.True(t, proto.Equal(&planetv1.WholeMap{
		StartTileId: 3, Codes: []string{"", "de", "fr"}, Tiles: []byte{2, 0, 0, 0, 0, 0, 1, 0},
	}, whole), "%v", whole)
}

func TestAWholeMapWithNothingHeldHasNoTile(t *testing.T) {
	whole := planetmessage.Whole(map[uint32]ledger.Owner{4: {}})

	assert.True(t, proto.Equal(&planetv1.WholeMap{Codes: []string{""}}, whole), "%v", whole)
}

func TestAWholeFrameIsWrittenWholeAndAFrameOfChangesTileByTile(t *testing.T) {
	at := time.Date(2026, 11, 18, 12, 0, 0, 0, time.UTC)

	history := planetmessage.MapHistory([]ledger.Frame{
		{At: at, Whole: true, Owners: map[uint32]ledger.Owner{1: {Country: "fr"}}},
		{At: at.Add(time.Hour), Owners: map[uint32]ledger.Owner{2: {Country: "de", At: at.Add(time.Minute)}}},
	}, nil, nil)

	assert.Equal(t, []byte{1, 0}, history.GetFrames()[0].GetWhole().GetTiles())
	assert.Empty(t, history.GetFrames()[0].GetOwners())
	assert.Nil(t, history.GetFrames()[1].GetWhole())
	assert.Equal(t, uint32(60), history.GetFrames()[1].GetOwners()[0].GetSecondsIntoStep())
}
