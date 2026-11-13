package spread_click_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/charges"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click_usecase/spread_click"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/notices"
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

const caller charges.Holder = "a-guest"

func played(t *testing.T) context.Context {
	t.Helper()

	return cpctx.AddAccountToContext(t.Context(), string(caller))
}

type stubNeighbours struct {
	around map[uint32][]uint32
	rings  []int
}

func (s *stubNeighbours) Around(_ string, tile uint32, rings int) []uint32 {
	s.rings = append(s.rings, rings)
	return slices.Clone(s.around[tile])
}

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

type recordingPublisher struct{ spreads []notices.Spread }

func (r *recordingPublisher) PublishSpread(spread notices.Spread) {
	r.spreads = append(r.spreads, spread)
}

func honeycomb() *stubNeighbours {
	return &stubNeighbours{around: map[uint32][]uint32{
		100: {90, 91, 99, 101, 109, 110},
		7:   nil,
	}}
}

type fixture struct {
	useCase   *spread_click.UseCase
	storage   *recordingStorage
	publisher *recordingPublisher
	hands     *charges.MemoryStore
	reach     *stubNeighbours
}

func setup(t *testing.T, clicksLeft int, err error, rules func() charges.Rules) *fixture {
	t.Helper()

	f := &fixture{
		storage:   &recordingStorage{tiles: map[uint32]string{}},
		publisher: &recordingPublisher{},
		hands:     charges.NewMemoryStore(),
		reach:     honeycomb(),
	}
	f.hands.Keep(caller, charges.Held{SpreadClicks: clicksLeft})
	f.useCase = spread_click.New(stubClick{storage: f.storage, err: err},
		charges.NewCharges(f.hands, rules), f.reach, f.storage, f.publisher)

	return f
}

func spreading(rings int, unlimited bool) func() charges.Rules {
	return func() charges.Rules {
		rules := charges.TestRules()
		rules.Spread.Rings, rules.Spread.Unlimited = rings, unlimited
		return rules
	}
}

func TestASpreadingClickTakesTheTileAndEveryTileTouchingIt(t *testing.T) {
	f := setup(t, 8, nil, charges.TestRules)

	_, err := f.useCase.Execute(played(t), click_usecase.In{TileID: 100, CountryID: "fr", Spread: true})
	require.NoError(t, err)

	assert.Equal(t, map[uint32]string{
		100: "fr", 90: "fr", 91: "fr", 99: "fr", 101: "fr", 109: "fr", 110: "fr",
	}, f.storage.tiles)
}

func TestTheSpreadReachesTheRingsInForce(t *testing.T) {
	f := setup(t, 8, nil, spreading(2, false))

	_, err := f.useCase.Execute(played(t), click_usecase.In{TileID: 100, CountryID: "fr", Spread: true})
	require.NoError(t, err)

	assert.Equal(t, []int{2}, f.reach.rings)
}

func TestASpreadTheWriterFailsIsAnErrorAndIsNotAnnounced(t *testing.T) {
	f := setup(t, 8, nil, charges.TestRules)
	f.storage.err = errors.New("out of range")

	_, err := f.useCase.Execute(played(t), click_usecase.In{TileID: 100, CountryID: "fr", Spread: true})

	require.ErrorIs(t, err, f.storage.err)
	assert.Empty(t, f.publisher.spreads)
}

func TestWithoutTheBonusAClickTakesOneTile(t *testing.T) {
	f := setup(t, 0, nil, charges.TestRules)

	_, err := f.useCase.Execute(played(t), click_usecase.In{TileID: 100, CountryID: "fr", Spread: true})
	require.NoError(t, err)

	assert.Equal(t, map[uint32]string{100: "fr"}, f.storage.tiles)
}

func TestARefusedClickSpreadsNothingAndCostsNoSpreadClick(t *testing.T) {
	refused := errors.New("unknown country")
	f := setup(t, 8, refused, charges.TestRules)

	_, err := f.useCase.Execute(played(t), click_usecase.In{TileID: 100, CountryID: "zz", Spread: true})

	require.ErrorIs(t, err, refused)
	assert.Empty(t, f.storage.tiles)
	assert.Empty(t, f.publisher.spreads)
	assert.Equal(t, 8, f.hands.Kept(caller).SpreadClicks)
}

func TestTheChargeSpreadsItsClicksAndNoMore(t *testing.T) {
	f := setup(t, 2, nil, charges.TestRules)

	for range 3 {
		_, err := f.useCase.Execute(played(t), click_usecase.In{TileID: 100, CountryID: "fr", Spread: true})
		require.NoError(t, err)
	}

	assert.Len(t, f.publisher.spreads, 2, "two clicks left spread two clicks, and the third takes one tile")
	assert.Zero(t, f.hands.Kept(caller).SpreadClicks)
}

func TestUnlimitedSpreadsSpendNoneOfTheSpreadClicksHeld(t *testing.T) {
	f := setup(t, 3, nil, spreading(2, true))

	for range 5 {
		_, err := f.useCase.Execute(played(t), click_usecase.In{TileID: 100, CountryID: "fr", Spread: true})
		require.NoError(t, err)
	}

	assert.Len(t, f.publisher.spreads, 5)
	assert.Equal(t, 3, f.hands.Kept(caller).SpreadClicks)
}

func TestUnlimitedSpreadsNeedNoAccountAndWaitForSpreadSwitchedOn(t *testing.T) {
	f := setup(t, 0, nil, spreading(1, true))

	_, err := f.useCase.Execute(t.Context(), click_usecase.In{TileID: 100, CountryID: "fr", Spread: true})
	require.NoError(t, err)
	_, err = f.useCase.Execute(t.Context(), click_usecase.In{TileID: 100, CountryID: "fr"})
	require.NoError(t, err)

	assert.Len(t, f.publisher.spreads, 1)
}

func TestACallerWithNoAccountSpreadsNothing(t *testing.T) {
	f := setup(t, 8, nil, charges.TestRules)

	_, err := f.useCase.Execute(t.Context(), click_usecase.In{TileID: 100, CountryID: "fr", Spread: true})
	require.NoError(t, err)

	assert.Equal(t, 8, f.hands.Kept(caller).SpreadClicks)
	assert.Len(t, f.storage.tiles, 1)
}

func TestALoneIslandTakesItselfAndNothingElse(t *testing.T) {
	f := setup(t, 8, nil, charges.TestRules)

	_, err := f.useCase.Execute(played(t), click_usecase.In{TileID: 7, CountryID: "fr", Spread: true})
	require.NoError(t, err)

	assert.Equal(t, map[uint32]string{7: "fr"}, f.storage.tiles)
}

func TestASpreadingClickIsAnnouncedWithTheTilesItTook(t *testing.T) {
	f := setup(t, 8, nil, charges.TestRules)

	_, err := f.useCase.Execute(played(t), click_usecase.In{TileID: 100, CountryID: "fr", Spread: true})
	require.NoError(t, err)

	assert.Equal(t, []notices.Spread{{
		CountryID:  "fr",
		Tile:       100,
		Neighbours: []uint32{90, 91, 99, 101, 109, 110},
	}}, f.publisher.spreads)
}

func TestNeitherAPlainNorARefusedClickIsAnnounced(t *testing.T) {
	plain := setup(t, 0, nil, charges.TestRules)
	_, err := plain.useCase.Execute(played(t), click_usecase.In{TileID: 100, CountryID: "fr", Spread: true})
	require.NoError(t, err)

	refused := setup(t, 8, errors.New("unknown country"), charges.TestRules)
	_, err = refused.useCase.Execute(played(t), click_usecase.In{TileID: 100, CountryID: "zz", Spread: true})
	require.Error(t, err)

	assert.Empty(t, plain.publisher.spreads)
	assert.Empty(t, refused.publisher.spreads)
}

func TestAClickWithSpreadSwitchedOffSpreadsNothingAndSpendsNothing(t *testing.T) {
	f := setup(t, 8, nil, charges.TestRules)

	_, err := f.useCase.Execute(played(t), click_usecase.In{TileID: 100, CountryID: "fr"})
	require.NoError(t, err)

	assert.Equal(t, map[uint32]string{100: "fr"}, f.storage.tiles)
	assert.Empty(t, f.publisher.spreads)
	assert.Equal(t, 8, f.hands.Kept(caller).SpreadClicks, "the pool is used only when the player chooses")
}

func TestASpreadTheStoreCannotSpendFailsWithTheTileTakenAndNoNeighbour(t *testing.T) {
	f := setup(t, 8, nil, charges.TestRules)
	down := errors.New("postgres is down")
	f.hands.FailWith(down)

	_, err := f.useCase.Execute(played(t), click_usecase.In{TileID: 100, CountryID: "fr", Spread: true})

	require.ErrorIs(t, err, down)
	assert.Equal(t, map[uint32]string{100: "fr"}, f.storage.tiles)
	assert.Empty(t, f.publisher.spreads)
}

type armReach map[string][]uint32

func (a armReach) Around(flag string, _ uint32, _ int) []uint32 { return a[flag] }

func TestASpreadTakesOnlyTheNeighboursInTheFlagsReach(t *testing.T) {
	storage := &recordingStorage{tiles: map[uint32]string{}}
	hands := charges.NewMemoryStore()
	hands.Keep(caller, charges.Held{SpreadClicks: 1})
	publisher := &recordingPublisher{}
	useCase := spread_click.New(stubClick{storage: storage}, charges.NewCharges(hands, charges.TestRules),
		armReach{"fr": {90, 91}}, storage, publisher)

	_, err := useCase.Execute(played(t), click_usecase.In{TileID: 100, CountryID: "fr", Spread: true})

	require.NoError(t, err)
	assert.Equal(t, map[uint32]string{100: "fr", 90: "fr", 91: "fr"}, storage.tiles)
	require.Len(t, publisher.spreads, 1)
	assert.Equal(t, []uint32{90, 91}, publisher.spreads[0].Neighbours)
}
