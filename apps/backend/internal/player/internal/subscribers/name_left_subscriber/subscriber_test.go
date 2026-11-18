package name_left_subscriber_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/pastnames"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/pastnames/inmemory_past_name_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/pastnames/usecases/keep_past_name_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/inmemory_player_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/subscribers/name_left_subscriber"
)

var (
	ada    = players.AccountID{15: 1}
	monday = time.Date(2026, 11, 16, 20, 0, 0, 0, time.UTC)
)

func subscriber(t *testing.T) (name_left_subscriber.Subscriber, *inmemory_past_name_store.Store) {
	t.Helper()

	profiles, kept := inmemory_player_store.New(), inmemory_past_name_store.New()
	require.NoError(t, profiles.SaveProfile(t.Context(), players.NewProfile(ada, "Ada", monday)))
	return name_left_subscriber.New(keep_past_name_usecase.New(profiles, kept)), kept
}

func TestTheNameLeftIsKept(t *testing.T) {
	handle, kept := subscriber(t)

	err := handle.Handle(t.Context(), &playerv1.NameLeft{AccountId: ada.String(), Name: "BraveFox42", LeftAt: timestamppb.New(monday)})

	require.NoError(t, err)
	assert.Equal(t, []pastnames.Snapshot{{Account: ada, Name: "BraveFox42", LeftAt: monday}}, kept.Kept())
}

func TestABrokenEventIsRefused(t *testing.T) {
	handle, kept := subscriber(t)

	assert.ErrorIs(t, handle.Handle(t.Context(), &playerv1.NameLeft{Name: "BraveFox42", LeftAt: timestamppb.New(monday)}),
		players.ErrInvalidAccount)
	assert.ErrorIs(t, handle.Handle(t.Context(), &playerv1.NameLeft{AccountId: ada.String(), LeftAt: timestamppb.New(monday)}),
		pastnames.ErrNoName)
	assert.ErrorIs(t, handle.Handle(t.Context(), &playerv1.NameLeft{AccountId: ada.String(), Name: "BraveFox42"}),
		pastnames.ErrNoTime)
	assert.Empty(t, kept.Kept())
}
