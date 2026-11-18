package titles_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/titles"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/titles/inmemory_title_store"
)

var (
	at  = time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC)
	ada = accounts.AccountID{15: 1}
)

func holding(t *testing.T, ids ...titles.ID) *inmemory_title_store.Store {
	t.Helper()

	store := inmemory_title_store.New()
	require.NoError(t, store.Grant(t.Context(), titles.Holdings{ada: ids}, at))
	return store
}

func TestShownIsWhatTheCatalogShowsOfTheTitlesHeld(t *testing.T) {
	shown, err := titles.NewBook(holding(t, "badge", "low", "mid"), tracks).Shown(t.Context(), ada)

	require.NoError(t, err)
	assert.Equal(t, []standingView{alone(badge), ranked(mid, ladder, 2)}, viewsOf(shown))
}
