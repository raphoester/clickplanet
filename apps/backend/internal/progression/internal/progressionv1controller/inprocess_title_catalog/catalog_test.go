package inprocess_title_catalog_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"

	progressionv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/progression/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/progressionv1controller/inprocess_title_catalog"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/progressionv1controller/progressionread"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/titles"
)

var (
	catalog = inprocess_title_catalog.New(titles.NewCatalog())
	raider  = &progressionv1.Title{
		Id: "raider", Name: "Raider", Rank: &progressionv1.Rank{TrackId: "conquest", TrackName: "Conquest", Number: 2, Count: 5},
		Feat: "Took 1,000 tiles",
	}
	loyal = &progressionv1.Title{
		Id: "loyal", Name: "Loyal", Rank: &progressionv1.Rank{TrackId: "devotion", TrackName: "Devotion", Number: 1, Count: 3},
		Feat: "Played 7 days in a row",
	}
	og = &progressionv1.Title{Id: "og", Name: "OG", Feat: "Joined before November 2026"}
)

func TestShownIsEachStandaloneTitleThenTheHighestRankOfEachTrack(t *testing.T) {
	shown := catalog.Shown([]string{"loyal", "raider", "settler", "og", "retired"})

	assert.True(t, proto.Equal(&progressionv1.GetTitlesResponse{Wearable: []*progressionv1.Title{og, raider, loyal}},
		&progressionv1.GetTitlesResponse{Wearable: shown}), "got %v", shown)
}

func TestWornIsTheChoiceOrTheFirstRankShown(t *testing.T) {
	held := []string{"loyal", "raider", "settler", "og"}

	assert.True(t, proto.Equal(loyal, catalog.Worn(held, "loyal")))
	assert.True(t, proto.Equal(raider, catalog.Worn(held, "settler")), "a lower rank chosen wears the rank shown on its track")
	assert.True(t, proto.Equal(raider, catalog.Worn(held, "")))
	assert.Nil(t, catalog.Worn(nil, "og"), "a choice not held is not worn")
}

func TestEachTrackIsMeasuredOnTheCareerAndSaysWhatIsHeld(t *testing.T) {
	tracks := catalog.Tracks([]string{"settler", "talker"}, progressionread.Career{
		TilesTaken: 250, StreakNow: 3, MessagesSent: 120, TilesFortified: 64, BombsDropped: 2, BonusesClaimed: 17, RightAnswers: 5,
		MissionsDone: 8,
	})

	progress := map[string]uint64{}
	for _, track := range tracks {
		progress[track.GetId()] = track.GetProgress()
	}
	assert.Equal(t, map[string]uint64{
		"conquest": 250, "devotion": 3, "chatter": 120, "defense": 64, "bombs": 2, "loot": 17, "quiz": 5, "missions": 8,
	}, progress)
	assert.True(t, tracks[0].GetSteps()[0].GetEarned())
	assert.False(t, tracks[0].GetSteps()[1].GetEarned(), "a rank is earned once granted, not because the career reached it")
	assert.Equal(t, uint64(1_000), tracks[0].GetSteps()[1].GetThreshold())
}
