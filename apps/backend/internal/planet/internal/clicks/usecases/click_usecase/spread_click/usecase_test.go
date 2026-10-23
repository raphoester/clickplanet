package spread_click_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click_usecase/spread_click"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
)

type stubClick struct {
	storage *recordingStorage
	err     error
}

func (s stubClick) Execute(ctx context.Context, in click_usecase.In) (click_usecase.Out, error) {
	if s.err != nil {
		return click_usecase.Out{}, s.err
	}

	return click_usecase.Out{}, s.storage.Set(ctx, in.TileID, in.CountryID)
}

const caller bonuses.Holder = "a-player"

func played(t *testing.T) context.Context {
	t.Helper()

	return cpctx.AddAccountToContext(t.Context(), string(caller))
}

type stubNeighbours map[uint32][]uint32

func (s stubNeighbours) Neighbours(id uint32) []uint32 { return s[id] }

type recordingStorage struct {
	tiles map[uint32]string
	err   error
}

func (r *recordingStorage) Set(_ context.Context, tile uint32, value string) error {
	r.tiles[tile] = value
	return nil
}

func (r *recordingStorage) Spread(_ context.Context, _ uint32, flag string, neighbours []uint32) error {
	if r.err != nil {
		return r.err
	}
	for _, neighbour := range neighbours {
		r.tiles[neighbour] = flag
	}
	return nil
}

type recordingPublisher struct{ spreads []bonuses.Spread }

func (r *recordingPublisher) PublishSpread(spread bonuses.Spread) {
	r.spreads = append(r.spreads, spread)
}

var honeycomb = stubNeighbours{
	100: {90, 91, 99, 101, 109, 110},
	7:   nil,
}

func setup(t *testing.T, spreading bool, err error) (*spread_click.UseCase, *recordingStorage) {
	t.Helper()

	useCase, storage, _ := setupWithPublisher(t, spreading, err)
	return useCase, storage
}

func setupWithPublisher(t *testing.T, spreading bool, err error) (*spread_click.UseCase, *recordingStorage, *recordingPublisher) {
	t.Helper()

	clicksLeft := 0
	if spreading {
		clicksLeft = 8
	}

	useCase, storage, publisher, _ := setupWithClicks(t, clicksLeft, err)

	return useCase, storage, publisher
}

func setupWithClicks(
	t *testing.T,
	clicksLeft int,
	err error,
) (*spread_click.UseCase, *recordingStorage, *recordingPublisher, *bonuses.MemoryStore) {
	t.Helper()

	storage := &recordingStorage{tiles: map[uint32]string{}}
	store := bonuses.NewMemoryStore()
	spreads := bonuses.NewCharges(store, bonuses.ChargesConfig{SpreadClicks: 8})
	if clicksLeft > 0 {
		require.NoError(t, spreads.Grant(t.Context(), caller, bonuses.KindSpreadClicks, clicksLeft))
	}
	publisher := &recordingPublisher{}
	useCase := spread_click.New(stubClick{storage: storage, err: err}, spreads, honeycomb, storage, publisher)

	return useCase, storage, publisher, store
}

func TestASpreadingClickTakesTheTileAndEveryTileTouchingIt(t *testing.T) {
	useCase, storage := setup(t, true, nil)

	_, err := useCase.Execute(played(t), click_usecase.In{TileID: 100, CountryID: "fr", Spread: true})
	require.NoError(t, err)

	assert.Equal(t, map[uint32]string{
		100: "fr", 90: "fr", 91: "fr", 99: "fr", 101: "fr", 109: "fr", 110: "fr",
	}, storage.tiles)
}

func TestASpreadTheWriterFailsIsAnErrorAndIsNotAnnounced(t *testing.T) {
	useCase, storage, publisher, _ := setupWithClicks(t, 8, nil)
	storage.err = errors.New("out of range")

	_, err := useCase.Execute(played(t), click_usecase.In{TileID: 100, CountryID: "fr", Spread: true})

	require.ErrorIs(t, err, storage.err)
	assert.Empty(t, publisher.spreads)
}

func TestWithoutTheBonusAClickTakesOneTile(t *testing.T) {
	useCase, storage := setup(t, false, nil)

	_, err := useCase.Execute(played(t), click_usecase.In{TileID: 100, CountryID: "fr", Spread: true})
	require.NoError(t, err)

	assert.Equal(t, map[uint32]string{100: "fr"}, storage.tiles)
}

func TestARefusedClickSpreadsNothingAndCostsNoSpreadClick(t *testing.T) {
	refused := errors.New("unknown country")
	useCase, storage, _, store := setupWithClicks(t, 8, refused)

	_, err := useCase.Execute(played(t), click_usecase.In{TileID: 100, CountryID: "zz", Spread: true})

	require.ErrorIs(t, err, refused)
	assert.Empty(t, storage.tiles)
	assert.Equal(t, 8, store.Kept(caller).SpreadClicks)
}

func TestTheChargeSpreadsItsClicksAndNoMore(t *testing.T) {
	useCase, _, publisher, store := setupWithClicks(t, 2, nil)

	for range 3 {
		_, err := useCase.Execute(played(t), click_usecase.In{TileID: 100, CountryID: "fr", Spread: true})
		require.NoError(t, err)
	}

	assert.Len(t, publisher.spreads, 2, "two clicks left spread two clicks, and the third takes one tile")
	assert.Zero(t, store.Kept(caller).SpreadClicks)
}

func TestTheSpreadSpentIsTheAccounts(t *testing.T) {
	useCase, storage, _, store := setupWithClicks(t, 8, nil)

	_, err := useCase.Execute(played(t), click_usecase.In{TileID: 100, CountryID: "fr", Spread: true})
	require.NoError(t, err)

	assert.Equal(t, 7, store.Kept(caller).SpreadClicks)
	assert.Len(t, storage.tiles, 7)
}

func TestACallerWithNoAccountSpreadsNothing(t *testing.T) {
	useCase, storage, publisher, store := setupWithClicks(t, 8, nil)

	_, err := useCase.Execute(t.Context(), click_usecase.In{TileID: 100, CountryID: "fr", Spread: true})
	require.NoError(t, err)

	assert.Equal(t, map[uint32]string{100: "fr"}, storage.tiles)
	assert.Empty(t, publisher.spreads)
	assert.Equal(t, 8, store.Kept(caller).SpreadClicks)
}

func TestASpreadTheStoreCannotSpendIsAnErrorAndTakesOneTile(t *testing.T) {
	useCase, storage, publisher, store := setupWithClicks(t, 8, nil)
	down := errors.New("postgres is down")
	store.FailWith(down)

	_, err := useCase.Execute(played(t), click_usecase.In{TileID: 100, CountryID: "fr", Spread: true})

	require.ErrorIs(t, err, down)
	assert.Equal(t, map[uint32]string{100: "fr"}, storage.tiles)
	assert.Empty(t, publisher.spreads)
}

func TestALoneIslandTakesItselfAndNothingElse(t *testing.T) {
	useCase, storage := setup(t, true, nil)

	_, err := useCase.Execute(played(t), click_usecase.In{TileID: 7, CountryID: "fr", Spread: true})
	require.NoError(t, err)

	assert.Equal(t, map[uint32]string{7: "fr"}, storage.tiles)
}

func TestASpreadingClickIsAnnouncedWithTheTilesItTook(t *testing.T) {
	useCase, _, publisher := setupWithPublisher(t, true, nil)

	_, err := useCase.Execute(played(t), click_usecase.In{TileID: 100, CountryID: "fr", Spread: true})
	require.NoError(t, err)

	assert.Equal(t, []bonuses.Spread{{
		CountryID:  "fr",
		Tile:       100,
		Neighbours: []uint32{90, 91, 99, 101, 109, 110},
	}}, publisher.spreads)
}

func TestTheAnnouncementDoesNotShareTheMapsTable(t *testing.T) {
	useCase, _, publisher := setupWithPublisher(t, true, nil)

	_, err := useCase.Execute(played(t), click_usecase.In{TileID: 100, CountryID: "fr", Spread: true})
	require.NoError(t, err)

	publisher.spreads[0].Neighbours[0] = 0
	assert.Equal(t, uint32(90), honeycomb[100][0])
}

func TestNeitherAPlainNorARefusedClickIsAnnounced(t *testing.T) {
	plain, _, quiet := setupWithPublisher(t, false, nil)
	_, err := plain.Execute(played(t), click_usecase.In{TileID: 100, CountryID: "fr", Spread: true})
	require.NoError(t, err)

	refused, _, refusedQuiet := setupWithPublisher(t, true, errors.New("unknown country"))
	_, err = refused.Execute(played(t), click_usecase.In{TileID: 100, CountryID: "zz", Spread: true})
	require.Error(t, err)

	assert.Empty(t, quiet.spreads)
	assert.Empty(t, refusedQuiet.spreads)
}

func TestAClickWithSpreadSwitchedOffSpreadsNothingAndSpendsNothing(t *testing.T) {
	useCase, storage, publisher, store := setupWithClicks(t, 8, nil)

	_, err := useCase.Execute(played(t), click_usecase.In{TileID: 100, CountryID: "fr"})
	require.NoError(t, err)

	assert.Equal(t, map[uint32]string{100: "fr"}, storage.tiles)
	assert.Empty(t, publisher.spreads)
	assert.Equal(t, 8, store.Kept(caller).SpreadClicks, "the pool is used only when the player chooses")
}
