package allegiance_click_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/allegiances"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
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

func setup(t *testing.T) (*allegiance_click.UseCase, *countingClicks, allegiances.Book, *cptime.FixedClock) {
	t.Helper()

	clock := cptime.NewFixedClock(time.Date(2026, 10, 17, 12, 0, 0, 0, time.UTC))
	inner := &countingClicks{}
	book := allegiances.NewBook(allegiances.NewMemoryStore(), cpcountries.New(), func() allegiances.Lock { return false }, clock)

	return allegiance_click.New(inner, book), inner, book, clock
}

func played(t *testing.T) context.Context {
	t.Helper()

	return cpctx.AddAccountToContext(t.Context(), alice)
}

func clickFor(t *testing.T, useCase *allegiance_click.UseCase, flag string) error {
	t.Helper()

	_, err := useCase.Execute(played(t), click_usecase.In{TileID: 7, CountryID: flag})
	return err
}

func TestAClickForTheFlagHeldReachesTheMap(t *testing.T) {
	useCase, inner, book, _ := setup(t)
	require.NoError(t, book.Pledge(played(t), clicks.PayerOf(played(t)), "fr"))

	require.NoError(t, clickFor(t, useCase, "fr"))
	require.NoError(t, clickFor(t, useCase, "fr"))

	assert.Len(t, inner.seen, 2)
}

func TestAClickWithNoFlagHeldReachesNothing(t *testing.T) {
	useCase, inner, _, _ := setup(t)

	err := clickFor(t, useCase, "fr")

	require.ErrorIs(t, err, allegiances.ErrNoFlag)
	assert.Empty(t, inner.seen)
}

func TestAClickForAnotherFlagReachesNothingAndChangesNoFlag(t *testing.T) {
	useCase, inner, book, clock := setup(t)
	require.NoError(t, book.Pledge(played(t), clicks.PayerOf(played(t)), "fr"))

	clock.Advance(2 * time.Hour)
	err := clickFor(t, useCase, "de")

	require.ErrorIs(t, err, allegiances.ErrOtherFlag, "with no wait running, only a pledge changes the flag")
	var refusal allegiances.Refusal
	require.ErrorAs(t, err, &refusal)
	assert.Equal(t, "fr", refusal.Flag)
	assert.Empty(t, inner.seen, "the throttle and the map are never asked")
	require.NoError(t, clickFor(t, useCase, "fr"))
}

func TestAClickWithNoAccountIsHeldToNothing(t *testing.T) {
	useCase, inner, _, _ := setup(t)

	_, err := useCase.Execute(t.Context(), click_usecase.In{TileID: 7, CountryID: "fr"})

	require.NoError(t, err)
	assert.Len(t, inner.seen, 1)
}
