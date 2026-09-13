package boost_click_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/bonus"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click/boost_click"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
)

type stubClick struct{ err error }

func (s stubClick) Execute(context.Context, click.In) (click.Out, error) {
	return click.Out{}, s.err
}

type stubBoosts struct{ keys map[string]bool }

func (s stubBoosts) Boosted(key string) bool { return s.keys[key] }

type recordingPublisher struct{ boosted []bonus.Boosted }

func (r *recordingPublisher) PublishBoosted(boosted bonus.Boosted) {
	r.boosted = append(r.boosted, boosted)
}

func setup(boosted bool, err error) (*boost_click.UseCase, *recordingPublisher) {
	boosts := stubBoosts{keys: map[string]bool{}}
	if boosted {
		boosts.keys[cpctx.RateLimitKey(context.Background())] = true
	}

	publisher := &recordingPublisher{}

	return boost_click.New(stubClick{err: err}, boosts, publisher), publisher
}

func TestABoostedClickIsAnnounced(t *testing.T) {
	useCase, publisher := setup(true, nil)

	_, err := useCase.Execute(t.Context(), click.In{TileID: 42, CountryID: "fr"})
	require.NoError(t, err)

	assert.Equal(t, []bonus.Boosted{{CountryID: "fr", Tile: 42}}, publisher.boosted)
}

func TestAPlainClickIsNotAnnounced(t *testing.T) {
	useCase, publisher := setup(false, nil)

	_, err := useCase.Execute(t.Context(), click.In{TileID: 42, CountryID: "fr"})
	require.NoError(t, err)

	assert.Empty(t, publisher.boosted)
}

func TestARefusedClickIsNotAnnounced(t *testing.T) {
	refused := errors.New("unknown country")
	useCase, publisher := setup(true, refused)

	_, err := useCase.Execute(t.Context(), click.In{TileID: 42, CountryID: "zz"})

	require.ErrorIs(t, err, refused)
	assert.Empty(t, publisher.boosted)
}
