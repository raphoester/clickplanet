package postgres_card_store_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/backgrounds"
	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/backgrounds/postgres_background_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/cards"
	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/cards/postgres_card_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg/cppgtest"
)

var server cppgtest.Server

func TestMain(m *testing.M) { server.Run(m) }

var (
	war    = backgrounds.BackgroundID{15: 1}
	points = backgrounds.BackgroundID{15: 2}
)

type row struct {
	Set      string
	Position int
	Headline string
	Line     string
}

func open(t *testing.T) cppg.Conn {
	t.Helper()
	db := server.CreateDatabase(t).OpenSchema(t, "promo", migrations.FS)
	for _, id := range []backgrounds.BackgroundID{war, points} {
		background, err := backgrounds.Keep(backgrounds.File{ID: id, MediaType: "image/png", Size: 1},
			id.String(), time.Now(), backgrounds.Rules{MaxBytes: 2 << 20})
		require.NoError(t, err)
		require.NoError(t, postgres_background_store.New(db).Keep(t.Context(), background))
	}
	return db
}

func save(t *testing.T, db cppg.Conn, set cards.SetName, drafts ...cards.Draft) error {
	t.Helper()
	deck, err := cards.NewDeck(set, drafts)
	require.NoError(t, err)
	return postgres_card_store.New(db).Save(t.Context(), deck) //nolint:wrapcheck // the test reads it.
}

func rows(t *testing.T, db cppg.Conn) []row {
	t.Helper()
	found, err := db.On(t.Context()).QueryContext(t.Context(),
		`SELECT rule_set, position, headline, line FROM cards ORDER BY rule_set, position`)
	require.NoError(t, err)
	defer func() { _ = found.Close() }()
	var out []row
	for found.Next() {
		var r row
		require.NoError(t, found.Scan(&r.Set, &r.Position, &r.Headline, &r.Line))
		out = append(out, r)
	}
	require.NoError(t, found.Err())
	return out
}

func TestASaveReplacesTheSetsCardsInOrder(t *testing.T) {
	db := open(t)
	require.NoError(t, save(t, db, "rush", cards.Draft{Background: war, Headline: "Rush"}))
	require.NoError(t, save(t, db, "urf", cards.Draft{Background: war, Headline: "Old"}, cards.Draft{Background: war, Headline: "Older"},
		cards.Draft{Background: war, Headline: "Oldest"}))

	require.NoError(t, save(t, db, "urf",
		cards.Draft{Background: points, Headline: "Points", Line: "×{clicks.tempo}"}, cards.Draft{Background: war, Headline: "War"}))

	assert.Equal(t, []row{
		{Set: "rush", Position: 0, Headline: "Rush"},
		{Set: "urf", Position: 0, Headline: "Points", Line: "×{clicks.tempo}"},
		{Set: "urf", Position: 1, Headline: "War"},
	}, rows(t, db))
}

func TestNoCardsRemovesThem(t *testing.T) {
	db := open(t)
	require.NoError(t, save(t, db, "urf", cards.Draft{Background: war, Headline: "War"}))

	require.NoError(t, save(t, db, "urf"))

	assert.Empty(t, rows(t, db))
}

func TestACardOnABackgroundNotKeptIsRefused(t *testing.T) {
	db := open(t)

	err := save(t, db, "urf", cards.Draft{Background: backgrounds.BackgroundID{15: 9}, Headline: "War"})

	require.ErrorIs(t, err, backgrounds.ErrNoBackground)
}
