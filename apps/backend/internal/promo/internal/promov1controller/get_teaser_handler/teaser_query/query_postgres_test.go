package teaser_query_test

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	promov1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/promo/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/backgrounds"
	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/backgrounds/postgres_background_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/cards"
	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/cards/postgres_card_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/promov1controller/get_teaser_handler/teaser_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

var (
	war    = backgrounds.BackgroundID{15: 1}
	points = backgrounds.BackgroundID{15: 2}
	urf    = &planetv1.Rules{
		Clicks:  &planetv1.ClickRules{Tempo: proto.Float64(3)},
		Bonuses: &planetv1.BonusRules{Boxes: &planetv1.BoxPace{MinInterval: durationpb.New(time.Minute)}},
	}
)

type battles map[string]string

func (b battles) SetOf(_ context.Context, battleID string) (string, error) {
	set, ok := b[battleID]
	if !ok {
		return "", fmt.Errorf("%w: %q", teaser_query.ErrNoBattle, battleID)
	}
	return set, nil
}

type planet map[string]*planetv1.Rules

func (p planet) Played(_ context.Context, set string) (*planetv1.Rules, error) {
	played, ok := p[set]
	if !ok {
		return nil, fmt.Errorf("%w: %q", cards.ErrUnknownSet, set)
	}
	return played, nil
}

func seeded(t *testing.T, decks map[cards.SetName][]cards.Draft) cppg.Conn {
	t.Helper()
	db := cppg.StartTestServer(t).OpenSchema(t, "promo", migrations.FS)
	for _, id := range []backgrounds.BackgroundID{war, points} {
		background, err := backgrounds.Keep(backgrounds.File{ID: id, MediaType: "image/png", Size: 1}, id.String(),
			time.Now(), backgrounds.Rules{MaxBytes: 2 << 20})
		require.NoError(t, err)
		require.NoError(t, postgres_background_store.New(db).Keep(t.Context(), background))
	}
	for set, drafts := range decks {
		deck, err := cards.NewDeck(set, drafts)
		require.NoError(t, err)
		require.NoError(t, postgres_card_store.New(db).Save(t.Context(), deck))
	}
	return db
}

func query(db cppg.Conn, log *bytes.Buffer) *teaser_query.PostgresQuery {
	return teaser_query.NewPostgresQuery(db,
		battles{"tonight": "urf", "season-1/finale": "urf", "calm": "calm", "gone": "gone"},
		planet{"urf": urf, "calm": urf},
		backgrounds.NewMemoryBucket(),
		slog.New(slog.NewTextHandler(log, nil)))
}

func url(id backgrounds.BackgroundID) string {
	return "https://media.example/promo/backgrounds/" + id.String()
}

func TestABattlesCardsAreFilledWithWhatItsSetPlays(t *testing.T) {
	db := seeded(t, map[cards.SetName][]cards.Draft{"urf": {
		{Background: war, Headline: "Clicks refill {clicks.tempo}× faster", Line: "Every country at once."},
		{Background: points, Headline: "A box every {bonuses.boxes.minInterval}"},
	}})

	for _, battle := range []string{"tonight", "season-1/finale"} {
		teaser, err := query(db, &bytes.Buffer{}).Teaser(t.Context(), battle)

		require.NoError(t, err)
		assert.True(t, proto.Equal(&promov1.GetTeaserResponse{Cards: []*promov1.TeaserCard{
			{BackgroundUrl: url(war), Headline: "Clicks refill 3× faster", Line: "Every country at once."},
			{BackgroundUrl: url(points), Headline: "A box every 1 min"},
		}}, teaser), "%s: %v", battle, teaser)
	}
}

func TestASetWithNoCardsTeasesNothing(t *testing.T) {
	teaser, err := query(seeded(t, nil), &bytes.Buffer{}).Teaser(t.Context(), "calm")

	require.NoError(t, err)
	assert.Empty(t, teaser.GetCards())
}

func TestABattleTheScheduleDoesNotHoldIsNone(t *testing.T) {
	_, err := query(seeded(t, nil), &bytes.Buffer{}).Teaser(t.Context(), "never")

	require.ErrorIs(t, err, teaser_query.ErrNoBattle)
}

func TestACardThatDoesNotFillIsLeftOutAndLogged(t *testing.T) {
	db := seeded(t, map[cards.SetName][]cards.Draft{"urf": {
		{Background: war, Headline: "Bombs blast {bonuses.bomb.rings} rings"},
		{Background: points, Headline: "Clicks refill {clicks.tempo}× faster"},
	}})
	var log bytes.Buffer

	teaser, err := query(db, &log).Teaser(t.Context(), "tonight")

	require.NoError(t, err)
	assert.True(t, proto.Equal(&promov1.GetTeaserResponse{Cards: []*promov1.TeaserCard{
		{BackgroundUrl: url(points), Headline: "Clicks refill 3× faster"},
	}}, teaser), "%v", teaser)
	assert.Contains(t, log.String(), "level=ERROR")
	assert.Contains(t, log.String(), `msg="a teaser card does not fill" set=urf card=1 battle=tonight`)
}

func TestASetPlanetNoLongerKeepsTeasesNothing(t *testing.T) {
	db := seeded(t, map[cards.SetName][]cards.Draft{"gone": {{Background: war, Headline: "War"}}})
	var log bytes.Buffer

	teaser, err := query(db, &log).Teaser(t.Context(), "gone")

	require.NoError(t, err)
	assert.Empty(t, teaser.GetCards())
	assert.Contains(t, log.String(), `msg="a teaser's set is no longer kept by planet" set=gone battle=gone`)
}
