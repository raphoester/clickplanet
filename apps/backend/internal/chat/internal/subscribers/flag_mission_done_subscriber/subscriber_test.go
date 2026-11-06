package flag_mission_done_subscriber_test

import (
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	missionsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/missions/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/announcements"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/announcements/inmemory_announcement_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/announcements/usecases/announce_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/feed/inprocess_feed"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/subscribers/flag_mission_done_subscriber"
)

var at = time.Date(2026, 11, 5, 12, 0, 0, 0, time.UTC)

func subscriber(store *inmemory_announcement_storage.Storage) flag_mission_done_subscriber.Subscriber {
	return flag_mission_done_subscriber.New(announce_usecase.New(store, inprocess_feed.New(0, slog.New(slog.DiscardHandler))))
}

func done(kind any) *missionsv1.FlagMissionDone {
	mission := &missionsv1.FlagMission{Id: "f1", Points: 6, Done: true}
	switch kind := kind.(type) {
	case *missionsv1.TakeIn:
		mission.Kind = &missionsv1.FlagMission_TakeIn{TakeIn: kind}
	case *missionsv1.BreakShields:
		mission.Kind = &missionsv1.FlagMission_BreakShields{BreakShields: kind}
	}
	return &missionsv1.FlagMissionDone{Flag: "fr", Mission: mission, DoneAt: timestamppb.New(at)}
}

func TestAFlagMissionDoneIsAnnouncedInItsFlagsRoomAndTheWorld(t *testing.T) {
	store := inmemory_announcement_storage.New()

	require.NoError(t, subscriber(store).Handle(t.Context(), done(&missionsv1.TakeIn{CountryId: "br", Goal: 3_000, Taken: 3_000})))

	kept := store.Kept()
	require.Len(t, kept, 1)
	assert.Equal(t, announcements.KindFlagMission, kept[0].Kind())
	assert.Equal(t, at, kept[0].At())
	assert.Equal(t, messages.FlagRoom("fr"), kept[0].Room())
	assert.JSONEq(t, `{"country":"fr","points":6,"mission":"take_in","ground":"br","goal":3000}`, string(kept[0].Payload()))
}

func TestAShieldMissionNamesTheRival(t *testing.T) {
	store := inmemory_announcement_storage.New()

	require.NoError(t, subscriber(store).Handle(t.Context(), done(&missionsv1.BreakShields{CountryId: "de", Goal: 300, Broken: 300})))

	require.Len(t, store.Kept(), 1)
	assert.JSONEq(t, `{"country":"fr","points":6,"mission":"break_shields","rival":"de","goal":300}`,
		string(store.Kept()[0].Payload()))
}

func TestAKindTheChatDoesNotKnowIsStillToldAsAMissionDone(t *testing.T) {
	store := inmemory_announcement_storage.New()

	require.NoError(t, subscriber(store).Handle(t.Context(), done(nil)))

	require.Len(t, store.Kept(), 1)
	assert.JSONEq(t, `{"country":"fr","points":6}`, string(store.Kept()[0].Payload()))
}

func TestAMissionToldTwiceIsAnnouncedOnce(t *testing.T) {
	store := inmemory_announcement_storage.New()
	told := done(&missionsv1.TakeIn{CountryId: "br", Goal: 3_000, Taken: 3_000})

	require.NoError(t, subscriber(store).Handle(t.Context(), told))
	require.NoError(t, subscriber(store).Handle(t.Context(), told))

	assert.Len(t, store.Kept(), 1)
}

func TestAMissionWithNoFlagNoIDOrNoTimeIsRefused(t *testing.T) {
	for name, broken := range map[string]func(*missionsv1.FlagMissionDone){
		"no flag": func(event *missionsv1.FlagMissionDone) { event.Flag = "" },
		"no id":   func(event *missionsv1.FlagMissionDone) { event.Mission.Id = "" },
		"no time": func(event *missionsv1.FlagMissionDone) { event.DoneAt = nil },
	} {
		store := inmemory_announcement_storage.New()
		event := done(nil)
		broken(event)

		require.Error(t, subscriber(store).Handle(t.Context(), event), name)
		assert.Empty(t, store.Kept(), name)
	}
}
