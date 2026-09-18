package publishing_drop_bomb_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses/usecases/drop_bomb_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses/usecases/drop_bomb_usecase/publishing_drop_bomb"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpbootstrap"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var now = time.Date(2026, 9, 18, 12, 0, 0, 0, time.UTC)

type stubUseCase struct {
	blast clicks.Blast
	err   error
}

func (s stubUseCase) Execute(context.Context, drop_bomb_usecase.In) (clicks.Blast, error) {
	return s.blast, s.err
}

type borders map[uint32]string

func (b borders) CountryOf(tile uint32) string { return b[tile] }

func TestABombOnLandIsPublishedWithTheGroundItHit(t *testing.T) {
	events := cpbootstrap.NewRecordedEvents()
	inner := stubUseCase{blast: clicks.Blast{Tile: 42, CountryID: "fr", Cleared: []uint32{41, 42, 43}}}

	_, err := publishing_drop_bomb.New(inner, borders{42: "de"}, events, cptime.NewFixedClock(now)).
		Execute(t.Context(), drop_bomb_usecase.In{CountryID: "fr"})

	require.NoError(t, err)
	require.Len(t, events.Published(), 1)
	assert.True(t, proto.Equal(&planetv1.BombLanded{
		Country: "fr", TileId: 42, Ground: "de", Cleared: 3, LandedAt: timestamppb.New(now),
	}, events.Published()[0]))
}

func TestABombInTheSeaIsPublishedWithNoGround(t *testing.T) {
	events := cpbootstrap.NewRecordedEvents()
	inner := stubUseCase{blast: clicks.Blast{CountryID: "fr"}}

	_, err := publishing_drop_bomb.New(inner, borders{0: "xx"}, events, cptime.NewFixedClock(now)).
		Execute(t.Context(), drop_bomb_usecase.In{CountryID: "fr"})

	require.NoError(t, err)
	require.Len(t, events.Published(), 1)
	assert.True(t, proto.Equal(&planetv1.BombLanded{Country: "fr", LandedAt: timestamppb.New(now)}, events.Published()[0]))
}

func TestARefusedDropOrADudPublishesNothing(t *testing.T) {
	events := cpbootstrap.NewRecordedEvents()
	refused := errors.New("no bomb")

	_, err := publishing_drop_bomb.New(stubUseCase{err: refused}, borders{}, events, cptime.NewFixedClock(now)).
		Execute(t.Context(), drop_bomb_usecase.In{CountryID: "fr"})
	require.ErrorIs(t, err, refused)

	_, err = publishing_drop_bomb.New(stubUseCase{}, borders{}, events, cptime.NewFixedClock(now)).
		Execute(t.Context(), drop_bomb_usecase.In{CountryID: "fr", Dud: true})
	require.NoError(t, err)

	assert.Empty(t, events.Published())
}
