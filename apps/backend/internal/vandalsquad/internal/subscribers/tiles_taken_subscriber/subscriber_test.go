package tiles_taken_subscriber_test

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
	"github.com/raphoester/clickplanet.lol-backend/internal/vandalsquad/internal/subscribers/tiles_taken_subscriber"
)

type window struct{ kept []paint.Take }

func (w *window) Keep(take paint.Take) { w.kept = append(w.kept, take) }

func TestEachTileOfATakeIsKeptInTheWindow(t *testing.T) {
	at := time.Date(2026, 11, 6, 20, 1, 0, 0, time.UTC)
	kept := &window{}

	require.NoError(t, tiles_taken_subscriber.New(kept).Handle(t.Context(), &planetv1.TilesTaken{
		AccountId: "01900000-0000-7000-8000-000000000001", Country: "ro", TakenAt: timestamppb.New(at),
		Tiles: []*planetv1.TakenTile{
			{TileId: 153032, Index: 0, Previous: "cz"},
			{TileId: 153033, Index: 1},
		},
	}))

	account := cpsession.AccountID(uuid.MustParse("01900000-0000-7000-8000-000000000001"))
	assert.Equal(t, []paint.Take{
		{Account: account, Tile: 153032, Flag: "ro", At: at},
		{Account: account, Tile: 153033, Flag: "ro", At: at},
	}, kept.kept)
}

func TestATakeOfNobodyIsNotKeptAndOneOfANonsenseAccountIsRefused(t *testing.T) {
	kept := &window{}
	subscriber := tiles_taken_subscriber.New(kept)
	tiles := []*planetv1.TakenTile{{TileId: 1}}

	require.NoError(t, subscriber.Handle(t.Context(), &planetv1.TilesTaken{Country: "ro", Tiles: tiles}))
	require.Error(t, subscriber.Handle(t.Context(), &planetv1.TilesTaken{AccountId: "nobody", Country: "ro", Tiles: tiles}))
	assert.Empty(t, kept.kept)
}
