package inprocess_title_catalog_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/inprocess_title_catalog"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/playerread"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/titles"
)

var (
	catalog = inprocess_title_catalog.New(titles.NewCatalog())
	raider  = &playerv1.Title{
		Id: "raider", Name: "Raider", Rank: &playerv1.Rank{TrackId: "conquest", TrackName: "Conquest", Number: 2, Count: 5},
	}
	loyal = &playerv1.Title{
		Id: "loyal", Name: "Loyal", Rank: &playerv1.Rank{TrackId: "devotion", TrackName: "Devotion", Number: 1, Count: 3},
	}
	og = &playerv1.Title{Id: "og", Name: "OG"}
)

func TestShownIsEachStandaloneTitleThenTheHighestRankOfEachTrack(t *testing.T) {
	shown := catalog.Shown([]string{"loyal", "raider", "settler", "og", "retired"})

	assert.True(t, proto.Equal(&playerv1.GetTitlesResponse{Wearable: []*playerv1.Title{og, raider, loyal}},
		&playerv1.GetTitlesResponse{Wearable: shown}), "got %v", shown)
}

func TestWornIsTheChoiceOrTheFirstRankShown(t *testing.T) {
	held := []string{"loyal", "raider", "settler", "og"}

	assert.True(t, proto.Equal(loyal, catalog.Worn(held, "loyal")))
	assert.True(t, proto.Equal(raider, catalog.Worn(held, "settler")), "a lower rank chosen wears the rank shown on its track")
	assert.True(t, proto.Equal(raider, catalog.Worn(held, "")))
	assert.Nil(t, catalog.Worn(nil, "og"), "a choice not held is not worn")
}

func TestEachTrackIsMeasuredOnTheCareerAndSaysWhatIsHeld(t *testing.T) {
	tracks := catalog.Tracks([]string{"settler", "talker"}, playerread.Career{TilesTaken: 250, StreakNow: 3, MessagesSent: 120})

	progress := map[string]uint64{}
	for _, track := range tracks {
		progress[track.GetId()] = track.GetProgress()
	}
	assert.Equal(t, map[string]uint64{"conquest": 250, "devotion": 3, "chatter": 120}, progress)
	assert.True(t, tracks[0].GetSteps()[0].GetEarned())
	assert.False(t, tracks[0].GetSteps()[1].GetEarned(), "a rank is earned once granted, not because the career reached it")
	assert.Equal(t, uint64(1_000), tracks[0].GetSteps()[1].GetThreshold())
}
