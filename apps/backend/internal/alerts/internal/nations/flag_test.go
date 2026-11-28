package nations_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/nations"
)

var at = time.Date(2026, 11, 28, 9, 0, 0, 0, time.UTC)

func TestAFlagWithARoleIsOneTheGameShipsWithOrASoldOneNamed(t *testing.T) {
	catalog := nations.NewCatalog(map[nations.Flag]string{"r-jz9cwqzp0k": "Heiliges Römisches Reich"})

	for _, code := range []string{"ie", "r-jz9cwqzp0k"} {
		flag, ok := catalog.FlagOf(code)
		require.True(t, ok, code)
		assert.Equal(t, nations.Flag(code), flag)
	}
	for _, code := range []string{"", "IE", "zz", "ireland", "r-k3p9x2ma0b", "xb"} {
		_, ok := catalog.FlagOf(code)
		assert.False(t, ok, code)
	}
}

func TestAFlagTheGameShipsWithIsNamedInEnglishAndASoldOneByItsName(t *testing.T) {
	catalog := nations.NewCatalog(map[nations.Flag]string{"r-jz9cwqzp0k": "Heiliges Römisches Reich", "gb-eng": "England"})

	for code, name := range map[nations.Flag]string{
		"ie":           "Ireland",
		"gb":           "United Kingdom",
		"us":           "United States",
		"pw":           "Palau",
		"xk":           "Kosovo",
		"r-jz9cwqzp0k": "Heiliges Römisches Reich",
		"gb-eng":       "England",
		"r-k3p9x2ma0b": "R-K3P9X2MA0B",
	} {
		assert.Equal(t, name, catalog.NameOf(code), code)
	}
}

func TestAFlagIsDrawnByItsTwoLettersOrItsNationsTags(t *testing.T) {
	assert.Equal(t, "🇮🇪", nations.Flag("ie").Emoji())
	assert.Equal(t, "🇵🇼", nations.Flag("pw").Emoji())
	assert.Equal(t, "🏴\U000E0067\U000E0062\U000E0065\U000E006E\U000E0067\U000E007F", nations.Flag("gb-eng").Emoji())
	assert.Equal(t, "🏴\U000E0067\U000E0062\U000E0073\U000E0063\U000E0074\U000E007F", nations.Flag("gb-sct").Emoji())
	assert.Empty(t, nations.Flag("xb").Emoji(), "Brittany's two letters are no ISO code")
	assert.Empty(t, nations.Flag("r-jz9cwqzp0k").Emoji())

	assert.Equal(t, "🇮🇪 Ireland", nations.Label("ie", "Ireland"))
	assert.Equal(t, "Heiliges Römisches Reich", nations.Label("r-jz9cwqzp0k", "Heiliges Römisches Reich"))
}

func TestTheLatestNameHoldsAndTheRestatementPutsOneRight(t *testing.T) {
	store := nations.NewInMemoryStore()
	catalog, err := nations.Load(t.Context(), store)
	require.NoError(t, err)

	flag, changed, err := catalog.Name(t.Context(), "r-jz9cwqzp0k", "Holy Roman Empire", at)
	require.NoError(t, err)
	assert.True(t, changed, "first heard")
	assert.Equal(t, nations.Flag("r-jz9cwqzp0k"), flag)

	_, changed, err = catalog.Name(t.Context(), "r-jz9cwqzp0k", "Holy Roman Empire", at)
	require.NoError(t, err)
	assert.False(t, changed, "told again")

	_, changed, err = catalog.Name(t.Context(), "r-jz9cwqzp0k", "Heiliges Römisches Reich", at.Add(time.Hour))
	require.NoError(t, err)
	assert.True(t, changed, "renamed")

	_, changed, err = catalog.Name(t.Context(), "r-jz9cwqzp0k", "Holy Roman Empire", at)
	require.NoError(t, err)
	assert.False(t, changed, "an opening heard after the rename it came before")
	assert.Equal(t, "Heiliges Römisches Reich", catalog.NameOf("r-jz9cwqzp0k"))

	_, changed, err = catalog.Name(t.Context(), "r-jz9cwqzp0k", "Sacrum Imperium", at.Add(time.Hour))
	require.NoError(t, err)
	assert.True(t, changed, "a name told at the time of the one kept replaces it: the restatement of a rename missed")

	reloaded, err := nations.Load(t.Context(), store)
	require.NoError(t, err)
	assert.Equal(t, "Sacrum Imperium", reloaded.NameOf("r-jz9cwqzp0k"), "kept for the next boot")

	_, _, err = catalog.Name(t.Context(), "fr", "France", at)
	require.ErrorIs(t, err, nations.ErrNotSold)
}
