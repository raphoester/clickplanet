package audit_paint_random_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/paint_random_tiles_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/paint_random_tiles_usecase/audit_paint_random"
)

type stubUseCase struct {
	out paint_random_tiles_usecase.Out
	err error
}

func (s stubUseCase) Execute(context.Context, paint_random_tiles_usecase.In) (paint_random_tiles_usecase.Out, error) {
	return s.out, s.err
}

func audited(inner stubUseCase) (*audit_paint_random.Audited, *bytes.Buffer) {
	var logs bytes.Buffer
	return audit_paint_random.New(inner, slog.New(slog.NewTextHandler(&logs, nil))), &logs
}

func TestAPaintIsLoggedWithItsCounts(t *testing.T) {
	want := paint_random_tiles_usecase.Out{Eligible: 9000, Picked: 500, OutsideArea: 40, Painted: 498}
	useCase, logs := audited(stubUseCase{out: want})

	out, err := useCase.Execute(t.Context(),
		paint_random_tiles_usecase.In{Flag: "dz", Area: "fr", Count: 500, Proximity: 0.8, DryRun: true})
	require.NoError(t, err)

	assert.Equal(t, want, out)
	assert.Contains(t, logs.String(),
		`level=WARN msg="admin random paint" flag=dz area=fr count=500 proximity=0.8 dryRun=true eligible=9000 picked=500 outsideArea=40 painted=498`)
}

func TestAFailureIsLoggedWithHowFarItGot(t *testing.T) {
	cause := errors.New("paint interrupted")
	useCase, logs := audited(stubUseCase{out: paint_random_tiles_usecase.Out{Painted: 256}, err: cause})

	out, err := useCase.Execute(t.Context(), paint_random_tiles_usecase.In{Flag: "dz", Area: "fr", Count: 500})

	require.ErrorIs(t, err, cause)
	assert.Equal(t, 256, out.Painted)
	assert.Contains(t, logs.String(), `msg="admin random paint failed"`)
	assert.Contains(t, logs.String(), `error="paint interrupted"`)
}
