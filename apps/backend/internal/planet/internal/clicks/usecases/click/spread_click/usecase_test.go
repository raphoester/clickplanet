package spread_click_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click/spread_click"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
)

// stubClick stands in for the rule, and writes the clicked tile the way it does.
type stubClick struct {
	storage *recordingStorage
	err     error
}

func (s stubClick) Execute(ctx context.Context, in click.In) (click.Out, error) {
	if s.err != nil {
		return click.Out{}, s.err
	}

	return click.Out{}, s.storage.Set(ctx, in.TileID, in.CountryID)
}

type stubSpreads struct{ scopes map[string]bool }

func (s stubSpreads) Spreading(scope string) bool { return s.scopes[scope] }

type stubNeighbours map[uint32][]uint32

func (s stubNeighbours) Neighbours(id uint32) []uint32 { return s[id] }

type recordingStorage struct{ tiles map[uint32]string }

func (r *recordingStorage) Set(_ context.Context, tile uint32, value string) error {
	r.tiles[tile] = value
	return nil
}

// A tile inland with its six neighbours, and a lone island with none.
var honeycomb = stubNeighbours{
	100: {90, 91, 99, 101, 109, 110},
	7:   nil,
}

func setup(spreading bool, err error) (*spread_click.UseCase, *recordingStorage) {
	storage := &recordingStorage{tiles: map[uint32]string{}}
	spreads := stubSpreads{scopes: map[string]bool{}}
	if spreading {
		spreads.scopes[cpctx.RateLimitKey(context.Background())] = true
	}

	return spread_click.New(stubClick{storage: storage, err: err}, spreads, honeycomb, storage), storage
}

func TestASpreadingClickTakesTheTileAndEveryTileTouchingIt(t *testing.T) {
	useCase, storage := setup(true, nil)

	_, err := useCase.Execute(t.Context(), click.In{TileID: 100, CountryID: "fr"})
	require.NoError(t, err)

	assert.Equal(t, map[uint32]string{
		100: "fr", 90: "fr", 91: "fr", 99: "fr", 101: "fr", 109: "fr", 110: "fr",
	}, storage.tiles)
}

func TestWithoutTheBonusAClickTakesOneTile(t *testing.T) {
	useCase, storage := setup(false, nil)

	_, err := useCase.Execute(t.Context(), click.In{TileID: 100, CountryID: "fr"})
	require.NoError(t, err)

	assert.Equal(t, map[uint32]string{100: "fr"}, storage.tiles)
}

func TestARefusedClickSpreadsNothing(t *testing.T) {
	refused := errors.New("unknown country")
	useCase, storage := setup(true, refused)

	_, err := useCase.Execute(t.Context(), click.In{TileID: 100, CountryID: "zz"})

	require.ErrorIs(t, err, refused)
	assert.Empty(t, storage.tiles)
}

func TestALoneIslandTakesItselfAndNothingElse(t *testing.T) {
	useCase, storage := setup(true, nil)

	_, err := useCase.Execute(t.Context(), click.In{TileID: 7, CountryID: "fr"})
	require.NoError(t, err)

	assert.Equal(t, map[uint32]string{7: "fr"}, storage.tiles)
}
