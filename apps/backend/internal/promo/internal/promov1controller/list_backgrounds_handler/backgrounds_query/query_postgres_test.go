package backgrounds_query_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	promov1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/promo/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/backgrounds"
	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/backgrounds/postgres_background_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/promov1controller/list_backgrounds_handler/backgrounds_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func TestTheLibraryIsListedNewestFirstWithItsPublicURLs(t *testing.T) {
	db := cppg.StartTestServer(t).OpenSchema(t, "promo", migrations.FS)
	at := time.Date(2026, 11, 5, 10, 0, 0, 0, time.UTC)
	war := backgrounds.File{ID: backgrounds.BackgroundID{15: 1}, MediaType: "image/webp", Size: 300_000}
	points := backgrounds.File{ID: backgrounds.BackgroundID{15: 2}, MediaType: "image/png", Size: 900_000}
	for i, file := range []backgrounds.File{war, points} {
		background, err := backgrounds.Keep(file, []string{"The war", "The points"}[i], at.Add(time.Duration(i)*time.Hour),
			backgrounds.Rules{MaxBytes: 2 << 20})
		require.NoError(t, err)
		require.NoError(t, postgres_background_store.New(db).Keep(t.Context(), background))
	}

	listed, err := backgrounds_query.NewPostgresQuery(db, backgrounds.NewMemoryBucket()).Backgrounds(t.Context())

	require.NoError(t, err)
	assert.True(t, proto.Equal(&promov1.ListBackgroundsResponse{Backgrounds: []*promov1.Background{
		{
			BackgroundId: points.ID.String(), Name: "The points", ContentType: "image/png", SizeBytes: 900_000,
			KeptAt: timestamppb.New(at.Add(time.Hour)), Url: "https://media.example/promo/backgrounds/" + points.ID.String(),
		},
		{
			BackgroundId: war.ID.String(), Name: "The war", ContentType: "image/webp", SizeBytes: 300_000,
			KeptAt: timestamppb.New(at), Url: "https://media.example/promo/backgrounds/" + war.ID.String(),
		},
	}}, listed), "%v", listed)
}
