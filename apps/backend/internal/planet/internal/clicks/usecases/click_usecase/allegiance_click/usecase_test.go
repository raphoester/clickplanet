package allegiance_click_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click_usecase/allegiance_click"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
)

type stubClick struct{ err error }

func (s stubClick) Execute(context.Context, click_usecase.In) (click_usecase.Out, error) {
	return click_usecase.Out{}, s.err
}

type recorded struct {
	payer   clicks.Payer
	country string
}

type recordingAllegiances struct{ clicks []recorded }

func (r *recordingAllegiances) Record(payer clicks.Payer, country string) {
	r.clicks = append(r.clicks, recorded{payer, country})
}

func TestAnAcceptedClickCountsForItsFlag(t *testing.T) {
	allegiances := &recordingAllegiances{}
	ctx := cpctx.AddAccountToContext(cpctx.AddIPToContext(t.Context(), "2001:db8::1"), "a-guest")

	_, err := allegiance_click.New(stubClick{}, allegiances).Execute(ctx, click_usecase.In{CountryID: "fr"})
	require.NoError(t, err)

	assert.Equal(t, []recorded{{clicks.PayerOf(ctx), "fr"}}, allegiances.clicks)
}

func TestARefusedClickCountsForNothing(t *testing.T) {
	allegiances := &recordingAllegiances{}

	_, err := allegiance_click.New(stubClick{err: errors.New("unknown country")}, allegiances).
		Execute(t.Context(), click_usecase.In{CountryID: "zz"})

	require.Error(t, err)
	assert.Empty(t, allegiances.clicks)
}
