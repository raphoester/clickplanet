package tile_taken_subscriber_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
	"github.com/raphoester/clickplanet.lol-backend/internal/vandalsquad/internal/paint"
	"github.com/raphoester/clickplanet.lol-backend/internal/vandalsquad/internal/subscribers/tile_taken_subscriber"
)

type window struct{ kept []paint.Take }

func (w *window) Keep(take paint.Take) { w.kept = append(w.kept, take) }

func TestATakeIsKeptInTheWindow(t *testing.T) {
	at := time.Date(2026, 11, 6, 20, 1, 0, 0, time.UTC)
	kept := &window{}

	require.NoError(t, tile_taken_subscriber.New(kept).Handle(t.Context(), &planetv1.TileTaken{
		AccountId: "01900000-0000-7000-8000-000000000001", TileId: 153032, Country: "ro", Previous: "cz",
		TakenAt: timestamppb.New(at),
	}))

	assert.Equal(t, []paint.Take{{
		Account: cpsession.AccountID(uuid.MustParse("01900000-0000-7000-8000-000000000001")),
		Tile:    153032, Flag: "ro", At: at,
	}}, kept.kept)
}

func TestATakeOfNobodyIsNotKeptAndOneOfANonsenseAccountIsRefused(t *testing.T) {
	kept := &window{}
	subscriber := tile_taken_subscriber.New(kept)

	require.NoError(t, subscriber.Handle(t.Context(), &planetv1.TileTaken{TileId: 1, Country: "ro"}))
	require.Error(t, subscriber.Handle(t.Context(), &planetv1.TileTaken{AccountId: "nobody", TileId: 1, Country: "ro"}))
	assert.Empty(t, kept.kept)
}
