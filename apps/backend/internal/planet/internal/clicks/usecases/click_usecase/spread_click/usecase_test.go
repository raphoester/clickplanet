package spread_click_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click_usecase/spread_click"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
)

// stubClick stands in for the rule, and writes the clicked tile the way it does.
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

// stubSpreads holds each holder's spread clicks left.
type stubSpreads struct{ left map[bonuses.Holder]int }

func (s stubSpreads) SpendSpreadClick(holder bonuses.Holder) bool {
	if s.left[holder] <= 0 {
		return false
	}
	s.left[holder]--

	return true
}

// caller is the holder of a click made from the test's context: no account, and the scope that reads.
var caller = bonuses.HolderOf(clicks.PayerOf(context.Background()))

type stubNeighbours map[uint32][]uint32

func (s stubNeighbours) Neighbours(id uint32) []uint32 { return s[id] }

type recordingStorage struct{ tiles map[uint32]string }

func (r *recordingStorage) Set(_ context.Context, tile uint32, value string) error {
	r.tiles[tile] = value
	return nil
}

type recordingPublisher struct{ spreads []bonuses.Spread }

func (r *recordingPublisher) PublishSpread(spread bonuses.Spread) {
	r.spreads = append(r.spreads, spread)
}

// A tile inland with its six neighbours, and a lone island with none.
var honeycomb = stubNeighbours{
	100: {90, 91, 99, 101, 109, 110},
	7:   nil,
}

func setup(spreading bool, err error) (*spread_click.UseCase, *recordingStorage) {
	useCase, storage, _ := setupWithPublisher(spreading, err)
	return useCase, storage
}

func setupWithPublisher(spreading bool, err error) (*spread_click.UseCase, *recordingStorage, *recordingPublisher) {
	clicksLeft := 0
	if spreading {
		clicksLeft = 8
	}

	useCase, storage, publisher, _ := setupWithClicks(clicksLeft, err)

	return useCase, storage, publisher
}

func setupWithClicks(clicksLeft int, err error) (*spread_click.UseCase, *recordingStorage, *recordingPublisher, stubSpreads) {
	storage := &recordingStorage{tiles: map[uint32]string{}}
	spreads := stubSpreads{left: map[bonuses.Holder]int{caller: clicksLeft}}
	publisher := &recordingPublisher{}

	return spread_click.New(stubClick{storage: storage, err: err}, spreads, honeycomb, storage, publisher), storage, publisher, spreads
}

func TestASpreadingClickTakesTheTileAndEveryTileTouchingIt(t *testing.T) {
	useCase, storage := setup(true, nil)

	_, err := useCase.Execute(t.Context(), click_usecase.In{TileID: 100, CountryID: "fr", Spread: true})
	require.NoError(t, err)

	assert.Equal(t, map[uint32]string{
		100: "fr", 90: "fr", 91: "fr", 99: "fr", 101: "fr", 109: "fr", 110: "fr",
	}, storage.tiles)
}

func TestWithoutTheBonusAClickTakesOneTile(t *testing.T) {
	useCase, storage := setup(false, nil)

	_, err := useCase.Execute(t.Context(), click_usecase.In{TileID: 100, CountryID: "fr", Spread: true})
	require.NoError(t, err)

	assert.Equal(t, map[uint32]string{100: "fr"}, storage.tiles)
}

func TestARefusedClickSpreadsNothingAndCostsNoSpreadClick(t *testing.T) {
	refused := errors.New("unknown country")
	useCase, storage, _, spreads := setupWithClicks(8, refused)

	_, err := useCase.Execute(t.Context(), click_usecase.In{TileID: 100, CountryID: "zz", Spread: true})

	require.ErrorIs(t, err, refused)
	assert.Empty(t, storage.tiles)
	assert.Equal(t, 8, spreads.left[caller])
}

func TestTheChargeSpreadsItsClicksAndNoMore(t *testing.T) {
	useCase, _, publisher, spreads := setupWithClicks(2, nil)

	for range 3 {
		_, err := useCase.Execute(t.Context(), click_usecase.In{TileID: 100, CountryID: "fr", Spread: true})
		require.NoError(t, err)
	}

	assert.Len(t, publisher.spreads, 2, "two clicks left spread two clicks, and the third takes one tile")
	assert.Zero(t, spreads.left[caller])
}

func TestTheSpreadSpentIsTheAccounts(t *testing.T) {
	storage := &recordingStorage{tiles: map[uint32]string{}}
	account := bonuses.HolderOf(clicks.Payer{Account: "a-guest"})
	spreads := stubSpreads{left: map[bonuses.Holder]int{account: 8}}
	useCase := spread_click.New(stubClick{storage: storage}, spreads, honeycomb, storage, &recordingPublisher{})

	_, err := useCase.Execute(cpctx.AddAccountToContext(t.Context(), "a-guest"), click_usecase.In{TileID: 100, CountryID: "fr", Spread: true})
	require.NoError(t, err)

	assert.Equal(t, 7, spreads.left[account])
	assert.Len(t, storage.tiles, 7)
}

func TestALoneIslandTakesItselfAndNothingElse(t *testing.T) {
	useCase, storage := setup(true, nil)

	_, err := useCase.Execute(t.Context(), click_usecase.In{TileID: 7, CountryID: "fr", Spread: true})
	require.NoError(t, err)

	assert.Equal(t, map[uint32]string{7: "fr"}, storage.tiles)
}

func TestASpreadingClickIsAnnouncedWithTheTilesItTook(t *testing.T) {
	useCase, _, publisher := setupWithPublisher(true, nil)

	_, err := useCase.Execute(t.Context(), click_usecase.In{TileID: 100, CountryID: "fr", Spread: true})
	require.NoError(t, err)

	assert.Equal(t, []bonuses.Spread{{
		CountryID:  "fr",
		Tile:       100,
		Neighbours: []uint32{90, 91, 99, 101, 109, 110},
	}}, publisher.spreads)
}

func TestTheAnnouncementDoesNotShareTheMapsTable(t *testing.T) {
	useCase, _, publisher := setupWithPublisher(true, nil)

	_, err := useCase.Execute(t.Context(), click_usecase.In{TileID: 100, CountryID: "fr", Spread: true})
	require.NoError(t, err)

	publisher.spreads[0].Neighbours[0] = 0
	assert.Equal(t, uint32(90), honeycomb[100][0])
}

func TestNeitherAPlainNorARefusedClickIsAnnounced(t *testing.T) {
	plain, _, quiet := setupWithPublisher(false, nil)
	_, err := plain.Execute(t.Context(), click_usecase.In{TileID: 100, CountryID: "fr", Spread: true})
	require.NoError(t, err)

	refused, _, refusedQuiet := setupWithPublisher(true, errors.New("unknown country"))
	_, err = refused.Execute(t.Context(), click_usecase.In{TileID: 100, CountryID: "zz", Spread: true})
	require.Error(t, err)

	assert.Empty(t, quiet.spreads)
	assert.Empty(t, refusedQuiet.spreads)
}

func TestAClickWithSpreadSwitchedOffSpreadsNothingAndSpendsNothing(t *testing.T) {
	useCase, storage, publisher, spreads := setupWithClicks(8, nil)

	_, err := useCase.Execute(t.Context(), click_usecase.In{TileID: 100, CountryID: "fr"})
	require.NoError(t, err)

	assert.Equal(t, map[uint32]string{100: "fr"}, storage.tiles)
	assert.Empty(t, publisher.spreads)
	assert.Equal(t, 8, spreads.left[caller], "the pool is used only when the player chooses")
}
