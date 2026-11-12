package postgres_background_store_test

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
	at    = time.Date(2026, 11, 5, 10, 0, 0, 0, time.UTC)
	rules = backgrounds.Rules{MaxBytes: 2 << 20}
	war   = backgrounds.File{ID: backgrounds.BackgroundID{15: 1}, MediaType: "image/webp", Size: 300_000}
)

func kept(t *testing.T, store *postgres_background_store.Store, file backgrounds.File, name string) {
	t.Helper()
	background, err := backgrounds.Keep(file, name, at, rules)
	require.NoError(t, err)
	require.NoError(t, store.Keep(t.Context(), background))
}

func names(t *testing.T, db cppg.Conn) []string {
	t.Helper()
	rows, err := db.On(t.Context()).QueryContext(t.Context(), `SELECT name FROM backgrounds ORDER BY name`)
	require.NoError(t, err)
	defer func() { _ = rows.Close() }()
	var found []string
	for rows.Next() {
		var name string
		require.NoError(t, rows.Scan(&name))
		found = append(found, name)
	}
	require.NoError(t, rows.Err())
	return found
}

func TestABackgroundKeptAgainTakesItsNewName(t *testing.T) {
	db := server.CreateDatabase(t).OpenSchema(t, "promo", migrations.FS)
	store := postgres_background_store.New(db)

	kept(t, store, war, "War")
	kept(t, store, war, "The war")

	assert.Equal(t, []string{"The war"}, names(t, db))
}

func TestABackgroundRemovedIsGone(t *testing.T) {
	db := server.CreateDatabase(t).OpenSchema(t, "promo", migrations.FS)
	store := postgres_background_store.New(db)
	kept(t, store, war, "The war")

	require.NoError(t, store.Remove(t.Context(), war.ID))

	assert.Empty(t, names(t, db))
	require.ErrorIs(t, store.Remove(t.Context(), war.ID), backgrounds.ErrNoBackground)
}

func TestABackgroundACardShowsStaysAndTheSetsAreNamed(t *testing.T) {
	db := server.CreateDatabase(t).OpenSchema(t, "promo", migrations.FS)
	store := postgres_background_store.New(db)
	kept(t, store, war, "The war")
	for _, set := range []cards.SetName{"urf", "rush"} {
		deck, err := cards.NewDeck(set, []cards.Draft{{Background: war.ID, Headline: "War"}})
		require.NoError(t, err)
		require.NoError(t, postgres_card_store.New(db).Save(t.Context(), deck))
	}

	err := store.Remove(t.Context(), war.ID)

	require.ErrorIs(t, err, backgrounds.ErrShown)
	assert.ErrorContains(t, err, "on the cards of rush, urf")
	assert.Equal(t, []string{"The war"}, names(t, db))
}
