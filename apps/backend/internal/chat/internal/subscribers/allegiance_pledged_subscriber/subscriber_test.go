package allegiance_pledged_subscriber_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/feed"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/feed/usecases/leave_room_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/subscribers/allegiance_pledged_subscriber"
)

type publisher struct {
	updates []feed.Update
}

func (p *publisher) Publish(update feed.Update) {
	p.updates = append(p.updates, update)
}

var ada = messages.AccountID{15: 1}

func heard(t *testing.T, event *planetv1.AllegiancePledged) ([]feed.Update, error) {
	t.Helper()

	updates := &publisher{}
	err := allegiance_pledged_subscriber.New(leave_room_usecase.New(updates)).Handle(t.Context(), event)
	return updates.updates, err //nolint:wrapcheck // the subscriber named it.
}

func TestAChangeOfFlagLeavesTheRoomOfTheFlagLeft(t *testing.T) {
	updates, err := heard(t, &planetv1.AllegiancePledged{AccountId: ada.String(), Country: "de", Previous: "fr"})

	require.NoError(t, err)
	assert.Equal(t, []feed.Update{feed.RoomLeft(ada, messages.FlagRoom("fr"))}, updates)
}

func TestAFirstFlagLeavesNoRoom(t *testing.T) {
	updates, err := heard(t, &planetv1.AllegiancePledged{AccountId: ada.String(), Country: "fr"})

	require.NoError(t, err)
	assert.Empty(t, updates)
}

func TestAPledgeOfNoAccountIsRefused(t *testing.T) {
	_, err := heard(t, &planetv1.AllegiancePledged{AccountId: "nobody", Country: "de", Previous: "fr"})

	require.Error(t, err)
}
