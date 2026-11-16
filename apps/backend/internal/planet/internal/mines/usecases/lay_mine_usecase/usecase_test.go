package lay_mine_usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/charges"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/mines/usecases/lay_mine_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/notices"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
)

const alice charges.Holder = "01926c6e-7a4b-7c3d-8e9f-0a1b2c3d4e5f"

var (
	aliceID = cpsession.AccountID{0x01, 0x92, 0x6c, 0x6e, 0x7a, 0x4b, 0x7c, 0x3d, 0x8e, 0x9f, 0x0a, 0x1b, 0x2c, 0x3d, 0x4e, 0x5f}
	laidAt  = clicks.MapVersion{Epoch: 7, Seq: 41}
)

type gate struct{ err error }

func (g gate) Lay(context.Context, clicks.Payer, uint32, string) error { return g.err }

type minefield struct {
	laid []clicks.MineSnapshot
	err  error
}

func (f *minefield) Lay(_ context.Context, mine clicks.Mine) (clicks.MapVersion, error) {
	f.laid = append(f.laid, mine.TakeSnapshot())
	return laidAt, f.err
}

type board struct{ told []notices.MineLaid }

func (b *board) PublishMineLaid(laid notices.MineLaid) { b.told = append(b.told, laid) }

type parts struct {
	hands     *charges.MemoryStore
	minefield *minefield
	board     *board
}

func setup(refused error, held charges.Held) (*lay_mine_usecase.UseCase, parts) {
	p := parts{hands: charges.NewMemoryStore(), minefield: &minefield{}, board: &board{}}
	p.hands.Keep(alice, held)
	return lay_mine_usecase.New(gate{err: refused}, charges.NewCharges(p.hands, charges.TestRules), p.minefield, p.board), p
}

func calling(t *testing.T) context.Context {
	t.Helper()
	return cpctx.AddAccountToContext(cpctx.AddIPToContext(t.Context(), "203.0.113.7"), string(alice))
}

func TestAMineLeavesTheHandForTheTileAndTheFlagsStreamsAloneHearOfIt(t *testing.T) {
	useCase, p := setup(nil, charges.Held{Mines: 2, Shields: 2})

	require.NoError(t, useCase.Execute(calling(t), lay_mine_usecase.In{TileID: 8, CountryID: "fr"}))

	assert.Equal(t, charges.Held{Mines: 1, Shields: 2}, p.hands.Kept(alice))
	assert.Equal(t, []clicks.MineSnapshot{{Tile: 8, Flag: "fr", Account: aliceID, Scope: "203.0.113.7"}}, p.minefield.laid)
	assert.Equal(t, []notices.MineLaid{{Flag: "fr", Tile: 8, Version: laidAt}}, p.board.told)
}

func TestAMineTheGateRefusesStaysInTheHand(t *testing.T) {
	useCase, p := setup(clicks.ErrNotMineable, charges.Held{Mines: 1})

	require.ErrorIs(t, useCase.Execute(calling(t), lay_mine_usecase.In{TileID: 9, CountryID: "fr"}), clicks.ErrNotMineable)

	assert.Equal(t, charges.Held{Mines: 1}, p.hands.Kept(alice))
	assert.Empty(t, p.minefield.laid)
	assert.Empty(t, p.board.told)
}

func TestNoMineHeldLaysNone(t *testing.T) {
	useCase, p := setup(nil, charges.Held{Bomb: true})

	require.ErrorIs(t, useCase.Execute(calling(t), lay_mine_usecase.In{TileID: 8, CountryID: "fr"}), lay_mine_usecase.ErrNoMine)
	require.ErrorIs(t, useCase.Execute(cpctx.AddIPToContext(t.Context(), "203.0.113.7"), lay_mine_usecase.In{TileID: 8, CountryID: "fr"}),
		lay_mine_usecase.ErrNoMine, "a caller with no account holds none")
	assert.Empty(t, p.minefield.laid)
}

func TestAMineTheMapRefusesAfterAllComesBackAndIsToldToNobody(t *testing.T) {
	useCase, p := setup(nil, charges.Held{Mines: 3})
	p.minefield.err = clicks.ErrNotMineable

	err := useCase.Execute(calling(t), lay_mine_usecase.In{TileID: 8, CountryID: "fr"})

	require.ErrorIs(t, err, clicks.ErrNotMineable)
	assert.Equal(t, charges.Held{Mines: 3}, p.hands.Kept(alice), "a full hand takes its mine back")
	assert.Empty(t, p.board.told)
}

func TestAWalletThatFailsFailsTheLayWithoutSayingThereIsNoMine(t *testing.T) {
	useCase, p := setup(nil, charges.Held{Mines: 1})
	down := errors.New("postgres is down")
	p.hands.FailWith(down)

	err := useCase.Execute(calling(t), lay_mine_usecase.In{TileID: 8, CountryID: "fr"})

	require.ErrorIs(t, err, down)
	assert.NotErrorIs(t, err, lay_mine_usecase.ErrNoMine)
	assert.Empty(t, p.minefield.laid)
}
