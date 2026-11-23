package lay_trap_usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/charges"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/notices"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/traps/usecases/lay_trap_usecase"
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

type trapline struct {
	laid []clicks.TrapSnapshot
	err  error
}

func (f *trapline) Lay(_ context.Context, trap clicks.Trap) (clicks.MapVersion, error) {
	f.laid = append(f.laid, trap.TakeSnapshot())
	return laidAt, f.err
}

type board struct{ told []notices.TrapLaid }

func (b *board) PublishTrapLaid(laid notices.TrapLaid) { b.told = append(b.told, laid) }

type parts struct {
	hands    *charges.MemoryStore
	trapline *trapline
	board    *board
}

func setup(refused error, held charges.Held) (*lay_trap_usecase.UseCase, parts) {
	p := parts{hands: charges.NewMemoryStore(), trapline: &trapline{}, board: &board{}}
	p.hands.Keep(alice, held)
	return lay_trap_usecase.New(gate{err: refused}, charges.NewCharges(p.hands, charges.TestRules), p.trapline, p.board), p
}

func calling(t *testing.T) context.Context {
	t.Helper()
	return cpctx.AddAccountToContext(cpctx.AddIPToContext(t.Context(), "203.0.113.7"), string(alice))
}

func TestATrapLeavesTheHandForTheTileAndTheFlagsStreamsAloneHearOfIt(t *testing.T) {
	useCase, p := setup(nil, charges.Held{Traps: 2, Shields: 2})

	require.NoError(t, useCase.Execute(calling(t), lay_trap_usecase.In{TileID: 8, CountryID: "fr"}))

	assert.Equal(t, charges.Held{Traps: 1, Shields: 2}, p.hands.Kept(alice))
	assert.Equal(t, []clicks.TrapSnapshot{{Tile: 8, Flag: "fr", Account: aliceID}}, p.trapline.laid)
	assert.Equal(t, []notices.TrapLaid{{Flag: "fr", Tile: 8, Version: laidAt}}, p.board.told)
}

func TestATrapTheGateRefusesStaysInTheHand(t *testing.T) {
	useCase, p := setup(clicks.ErrNotTrappable, charges.Held{Traps: 1})

	require.ErrorIs(t, useCase.Execute(calling(t), lay_trap_usecase.In{TileID: 9, CountryID: "fr"}), clicks.ErrNotTrappable)

	assert.Equal(t, charges.Held{Traps: 1}, p.hands.Kept(alice))
	assert.Empty(t, p.trapline.laid)
	assert.Empty(t, p.board.told)
}

func TestNoTrapHeldLaysNone(t *testing.T) {
	useCase, p := setup(nil, charges.Held{Bombs: 1})

	require.ErrorIs(t, useCase.Execute(calling(t), lay_trap_usecase.In{TileID: 8, CountryID: "fr"}), lay_trap_usecase.ErrNoTrap)
	require.ErrorIs(t, useCase.Execute(cpctx.AddIPToContext(t.Context(), "203.0.113.7"), lay_trap_usecase.In{TileID: 8, CountryID: "fr"}),
		lay_trap_usecase.ErrNoTrap, "a caller with no account holds none")
	assert.Empty(t, p.trapline.laid)
}

func TestATrapTheMapRefusesAfterAllComesBackAndIsToldToNobody(t *testing.T) {
	useCase, p := setup(nil, charges.Held{Traps: 3})
	p.trapline.err = clicks.ErrNotTrappable

	err := useCase.Execute(calling(t), lay_trap_usecase.In{TileID: 8, CountryID: "fr"})

	require.ErrorIs(t, err, clicks.ErrNotTrappable)
	assert.Equal(t, charges.Held{Traps: 3}, p.hands.Kept(alice), "a full hand takes its trap back")
	assert.Empty(t, p.board.told)
}

func TestAWalletThatFailsFailsTheLayWithoutSayingThereIsNoTrap(t *testing.T) {
	useCase, p := setup(nil, charges.Held{Traps: 1})
	down := errors.New("postgres is down")
	p.hands.FailWith(down)

	err := useCase.Execute(calling(t), lay_trap_usecase.In{TileID: 8, CountryID: "fr"})

	require.ErrorIs(t, err, down)
	assert.NotErrorIs(t, err, lay_trap_usecase.ErrNoTrap)
	assert.Empty(t, p.trapline.laid)
}
