package use_virus_usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/charges"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/viruses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/viruses/usecases/use_virus_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
)

const alice charges.Holder = "01926c6e-7a4b-7c3d-8e9f-0a1b2c3d4e5f"

var (
	aliceID = cpsession.AccountID{0x01, 0x92, 0x6c, 0x6e, 0x7a, 0x4b, 0x7c, 0x3d, 0x8e, 0x9f, 0x0a, 0x1b, 0x2c, 0x3d, 0x4e, 0x5f}
	noon    = time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)
)

type gate struct{ err error }

func (g gate) Aim(context.Context, clicks.Payer, uint32, string) error { return g.err }

type contagion struct {
	started []clicks.VirusSnapshot
	most    int
	next    time.Time
	err     error
}

func (c *contagion) Start(_ context.Context, virus clicks.Virus, most int, next time.Time) (clicks.Virus, error) {
	c.started = append(c.started, virus.TakeSnapshot())
	c.most, c.next = most, next
	return virus, c.err
}

type outbreak struct{}

func (outbreak) Most() int       { return 32 }
func (outbreak) Next() time.Time { return noon }

type parts struct {
	hands     *charges.MemoryStore
	contagion *contagion
}

func setup(aimed error, held charges.Held) (*use_virus_usecase.UseCase, parts) {
	p := parts{hands: charges.NewMemoryStore(), contagion: &contagion{}}
	p.hands.Keep(alice, held)
	return use_virus_usecase.New(gate{err: aimed}, &viruses.SequentialIDs{},
		charges.NewCharges(p.hands, charges.TestRules), p.contagion, outbreak{}), p
}

func calling(t *testing.T) context.Context {
	t.Helper()
	return cpctx.AddAccountToContext(cpctx.AddIPToContext(t.Context(), "203.0.113.7"), string(alice))
}

func TestAVirusLeavesTheHandAndStartsOnTheTileForTheCallersFlag(t *testing.T) {
	useCase, p := setup(nil, charges.Held{Viruses: 1, Shields: 2})

	require.NoError(t, useCase.Execute(calling(t), use_virus_usecase.In{TileID: 8, CountryID: "fr"}))

	assert.Equal(t, charges.Held{Shields: 2}, p.hands.Kept(alice))
	assert.Equal(t, []clicks.VirusSnapshot{{
		ID: "00000000-0000-7000-8000-000000000001", Account: aliceID, Scope: "203.0.113.7", Flag: "fr", Tile: 8,
	}}, p.contagion.started)
	assert.Equal(t, 32, p.contagion.most)
	assert.Equal(t, noon, p.contagion.next)
}

func TestAVirusTheGateRefusesStaysInTheHand(t *testing.T) {
	useCase, p := setup(clicks.ErrNotInfectable, charges.Held{Viruses: 1})

	err := useCase.Execute(calling(t), use_virus_usecase.In{TileID: 9, CountryID: "fr"})

	require.ErrorIs(t, err, clicks.ErrNotInfectable)
	assert.Equal(t, charges.Held{Viruses: 1}, p.hands.Kept(alice))
	assert.Empty(t, p.contagion.started)
}

func TestNoVirusHeldStartsNone(t *testing.T) {
	useCase, p := setup(nil, charges.Held{Bombs: 1})

	require.ErrorIs(t, useCase.Execute(calling(t), use_virus_usecase.In{TileID: 8, CountryID: "fr"}), use_virus_usecase.ErrNoVirus)
	require.ErrorIs(t, useCase.Execute(cpctx.AddIPToContext(t.Context(), "203.0.113.7"), use_virus_usecase.In{TileID: 8, CountryID: "fr"}),
		use_virus_usecase.ErrNoVirus, "a caller with no account holds none")
	assert.Empty(t, p.contagion.started)
}

func TestAVirusTheMapRefusesAfterAllComesBack(t *testing.T) {
	useCase, p := setup(nil, charges.Held{Viruses: 1})
	p.contagion.err = clicks.ErrNotInfectable

	err := useCase.Execute(calling(t), use_virus_usecase.In{TileID: 8, CountryID: "fr"})

	require.ErrorIs(t, err, clicks.ErrNotInfectable)
	assert.Equal(t, charges.Held{Viruses: 1}, p.hands.Kept(alice), "a shield placed meanwhile costs no virus")
}

func TestAWalletThatFailsFailsTheUseWithoutSayingThereIsNoVirus(t *testing.T) {
	useCase, p := setup(nil, charges.Held{Viruses: 1})
	down := errors.New("postgres is down")
	p.hands.FailWith(down)

	err := useCase.Execute(calling(t), use_virus_usecase.In{TileID: 8, CountryID: "fr"})

	require.ErrorIs(t, err, down)
	assert.NotErrorIs(t, err, use_virus_usecase.ErrNoVirus)
	assert.Empty(t, p.contagion.started)
}
