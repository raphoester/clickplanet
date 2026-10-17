package allegiance_click_test

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/allegiances"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/allegiances/inmemory_allegiance_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click_usecase/allegiance_click"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcountries"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

const alice = "01926c6e-7a4b-7c3d-8e9f-0a1b2c3d4e5f"

type countingClicks struct {
	seen []click_usecase.In
}

func (c *countingClicks) Execute(_ context.Context, in click_usecase.In) (click_usecase.Out, error) {
	c.seen = append(c.seen, in)
	return click_usecase.Out{}, nil
}

func setup(t *testing.T) (*allegiance_click.UseCase, *countingClicks, *cptime.FixedClock) {
	t.Helper()

	storage := inmemory_allegiance_storage.New(inmemory_allegiance_storage.Config{},
		inmemory_allegiance_storage.NewMemoryPersistence(), slog.New(slog.DiscardHandler))
	clock := cptime.NewFixedClock(time.Date(2026, 10, 17, 12, 0, 0, 0, time.UTC))
	inner := &countingClicks{}

	return allegiance_click.New(inner, allegiances.NewBook(storage, cpcountries.New(), clock)), inner, clock
}

func clickFor(t *testing.T, useCase *allegiance_click.UseCase, flag string) error {
	t.Helper()

	_, err := useCase.Execute(cpctx.AddAccountToContext(t.Context(), alice), click_usecase.In{TileID: 7, CountryID: flag})
	return err
}

func TestAClickForTheFlagHeldReachesTheMap(t *testing.T) {
	useCase, inner, _ := setup(t)

	require.NoError(t, clickFor(t, useCase, "fr"))
	require.NoError(t, clickFor(t, useCase, "fr"))

	assert.Len(t, inner.seen, 2)
}

func TestAClickForAnotherFlagWithinTheHourReachesNothing(t *testing.T) {
	useCase, inner, clock := setup(t)
	require.NoError(t, clickFor(t, useCase, "fr"))
	require.NoError(t, clickFor(t, useCase, "de"))

	clock.Advance(30 * time.Minute)
	err := clickFor(t, useCase, "fr")

	require.ErrorIs(t, err, allegiances.ErrChangedTooSoon)
	assert.Len(t, inner.seen, 2, "the throttle and the map are never asked")
}
