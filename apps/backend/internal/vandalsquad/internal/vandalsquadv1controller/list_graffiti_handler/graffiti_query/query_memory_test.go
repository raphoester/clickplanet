package graffiti_query_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	vandalsquadv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/vandalsquad/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
	"github.com/raphoester/clickplanet.lol-backend/internal/vandalsquad/internal/drawings"
	"github.com/raphoester/clickplanet.lol-backend/internal/vandalsquad/internal/graffiti"
	"github.com/raphoester/clickplanet.lol-backend/internal/vandalsquad/internal/vandalsquadv1controller/list_graffiti_handler/graffiti_query"
)

type book []graffiti.Graffiti

func (b book) Entries() []graffiti.Graffiti { return b }

func TestEachGraffitiIsListedAsTheBookKeepsIt(t *testing.T) {
	flagged := time.Date(2026, 11, 6, 20, 20, 0, 0, time.UTC)
	answer := graffiti_query.NewMemoryQuery(book{{
		Account: cpsession.AccountID(uuid.MustParse("01900000-0000-7000-8000-000000000001")),
		Flag:    "ro", Kind: graffiti.Swastika, Tiles: []drawings.TileID{4, 7},
		FlaggedAt: flagged, SeenAt: flagged.Add(time.Minute),
		Link: "https://clickplanet.lol/play?lat=-4.10&lon=22.20&zoom=20.0",
	}}).Graffiti()

	assert.True(t, proto.Equal(&vandalsquadv1.ListGraffitiResponse{Graffiti: []*vandalsquadv1.Graffiti{{
		AccountId: "01900000-0000-7000-8000-000000000001",
		CountryId: "ro", Kind: vandalsquadv1.GraffitiKind_GRAFFITI_KIND_SWASTIKA, TileIds: []uint32{4, 7},
		FlaggedAt: timestamppb.New(flagged), SeenAt: timestamppb.New(flagged.Add(time.Minute)),
		Link: "https://clickplanet.lol/play?lat=-4.10&lon=22.20&zoom=20.0",
	}}}, answer), answer)
}

func TestAnEmptyBookIsAnEmptyList(t *testing.T) {
	assert.Empty(t, graffiti_query.NewMemoryQuery(book{}).Graffiti().GetGraffiti())
}
