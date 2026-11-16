package detonate_mines_usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/mines/usecases/detonate_mines_usecase"
)

type minefield struct {
	rings       []int
	detonations []clicks.Detonation
	err         error
}

func (f *minefield) Detonate(_ context.Context, rings int) ([]clicks.Detonation, error) {
	f.rings = append(f.rings, rings)
	return f.detonations, f.err
}

type blasting struct{}

func (blasting) Rings() int { return 2 }

func TestEveryMineTrippedGoesOffAtTheBlastsWidthInForce(t *testing.T) {
	field := &minefield{detonations: []clicks.Detonation{{}, {}}}

	detonated, err := detonate_mines_usecase.New(field, blasting{}).Execute(t.Context())

	require.NoError(t, err)
	assert.Equal(t, 2, detonated)
	assert.Equal(t, []int{2}, field.rings)
}

func TestAFailureSaysHowManyWentOffBeforeIt(t *testing.T) {
	down := errors.New("tile out of range")
	field := &minefield{detonations: []clicks.Detonation{{}}, err: down}

	detonated, err := detonate_mines_usecase.New(field, blasting{}).Execute(t.Context())

	require.ErrorIs(t, err, down)
	assert.Equal(t, 1, detonated)
}
