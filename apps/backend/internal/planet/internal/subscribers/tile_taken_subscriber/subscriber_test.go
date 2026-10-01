package tile_taken_subscriber_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/inmemory_allegiance_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/record_allegiance_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/subscribers/tile_taken_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

const account = "0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11"

var at = time.Date(2026, 9, 17, 12, 0, 0, 0, time.UTC)

func setUp() (tile_taken_subscriber.Subscriber, *inmemory_allegiance_storage.Storage) {
	allegiances := inmemory_allegiance_storage.New(cptime.NewFixedClock(at))
	return tile_taken_subscriber.New(record_allegiance_usecase.New(allegiances)), allegiances
}

func TestATakeCountsForItsAccountAndItsScope(t *testing.T) {
	subscriber, allegiances := setUp()

	err := subscriber.Handle(t.Context(), &planetv1.TileTaken{
		AccountId: account, TileId: 42, Country: "fr", TakenAt: timestamppb.New(at), Scope: "2001:db8::/64",
	})

	require.NoError(t, err)
	assert.Equal(t, "fr", allegiances.Allegiance(clicks.AccountAllegianceKey(account)).Flag())
	assert.Equal(t, "fr", allegiances.Allegiance(clicks.ScopeAllegianceKey("2001:db8::/64")).Flag())
}

func TestAnEventWithNoCountryOrNoTimeIsRefused(t *testing.T) {
	subscriber, allegiances := setUp()

	require.Error(t, subscriber.Handle(t.Context(), &planetv1.TileTaken{AccountId: account, TileId: 42, TakenAt: timestamppb.New(at)}))
	require.Error(t, subscriber.Handle(t.Context(), &planetv1.TileTaken{AccountId: account, TileId: 42, Country: "fr"}))
	assert.Empty(t, allegiances.Allegiance(clicks.AccountAllegianceKey(account)).Flag())
}
