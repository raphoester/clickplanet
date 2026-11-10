package arena_opened_subscriber_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/arenas/arena_opened_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/arenas/inmemory_arena_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/arenas/usecases/hear_opened_arena_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments"
)

func TestAnArenaToldOpenedIsKeptOnce(t *testing.T) {
	store := inmemory_arena_store.New()
	subscriber := arena_opened_subscriber.New(hear_opened_arena_usecase.New(store))
	event := &planetv1.ArenaOpened{ArenaId: "season-1/finale", OpenedAt: timestamppb.Now()}

	require.NoError(t, subscriber.Handle(t.Context(), event))
	require.NoError(t, subscriber.Handle(t.Context(), event))
	require.NoError(t, subscriber.Handle(t.Context(), &planetv1.ArenaOpened{}))

	kept, err := store.ArenasKept(t.Context())
	require.NoError(t, err)
	assert.Equal(t, []moments.RoundID{"season-1/finale"}, kept.Opened)
	assert.Empty(t, kept.Closed)
}
