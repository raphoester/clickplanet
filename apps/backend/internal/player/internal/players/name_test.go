package players_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
)

func TestANameKeepsTheCaseItWasTypedIn(t *testing.T) {
	name, err := players.NameOf("Ada_Lovelace_18")

	require.NoError(t, err)
	assert.Equal(t, players.Name("Ada_Lovelace_18"), name)
}

func TestANameIsThreeToFifteenCharactersNotBytes(t *testing.T) {
	for _, value := range []string{"abc", strings.Repeat("a", players.MaxNameLength), strings.Repeat("é", players.MaxNameLength), "李小龍"} {
		_, err := players.NameOf(value)
		require.NoError(t, err, "%q", value)
	}

	for _, value := range []string{"", "ab", "李龍", strings.Repeat("a", players.MaxNameLength+1), strings.Repeat("é", players.MaxNameLength+1)} {
		_, err := players.NameOf(value)
		require.ErrorIs(t, err, players.ErrInvalidName, "%q", value)
	}
}

func TestANameTakesLettersOfAnyScriptDigitsUnderscoresAndSpaces(t *testing.T) {
	for _, value := range []string{
		"Ada Lovelace", "Émile Zola", "Жанна Д", "東京タワー", "محمد", "שלום", "Ελένη", "नमस्ते", "김민수", "Ken太郎", "ПУТНИК_7", "٣٤ علي",
	} {
		_, err := players.NameOf(value)
		require.NoError(t, err, "%q", value)
	}
}

func TestANameIsPutInNFC(t *testing.T) {
	name, err := players.NameOf("E\u0301mile")

	require.NoError(t, err)
	assert.Equal(t, players.Name("\u00c9mile"), name)
}

func TestTheSpacesAtBothEndsAreCutAndTwoInARowAreRefused(t *testing.T) {
	name, err := players.NameOf("  Ada L ")
	require.NoError(t, err)
	assert.Equal(t, players.Name("Ada L"), name)

	for _, value := range []string{"Ada  L", " a ", "   "} {
		_, err := players.NameOf(value)
		require.ErrorIs(t, err, players.ErrInvalidName, "%q", value)
	}
}

func TestANameRefusesEmojisPunctuationAndSymbols(t *testing.T) {
	for _, value := range []string{"🌍🌍🌍", "Ada🌍", "Ada-L", "Ada.L", "Ada!", "Ada@x", "Ada™", "Ada²", "Ada€", "a⃣bc", "Ada、L", string([]byte{0xff, 0xfe, 0xfd})} {
		_, err := players.NameOf(value)
		require.ErrorIs(t, err, players.ErrInvalidName, "%q", value)
	}
}

func TestANameRefusesControlsAndInvisibleCharacters(t *testing.T) {
	for _, value := range []string{
		"Ada\n", "Ada\tL", "Ada\u200bL", "Ada\u200dL", "Ada\u200cL", "\u202eAda", "Ada\u2066L", "Ada\ufeff", "Ada\u00adL",
		"Ada\u00a0L", "Ada\u3000L", "Ada\u3164", "Ada\ufe0f", "Ada\u2800",
	} {
		_, err := players.NameOf(value)
		require.ErrorIs(t, err, players.ErrInvalidName, "%q", value)
	}
}

func TestAMarkFollowsALetterAndNeverStacksHigh(t *testing.T) {
	for _, value := range []string{"\u0301Ada", "Ada \u0301", "Ad1\u0301", "Adx" + strings.Repeat("\u0301", 4)} {
		_, err := players.NameOf(value)
		require.ErrorIs(t, err, players.ErrInvalidName, "%q", value)
	}
}

func TestANameDoesNotMixScriptsThatLookAlike(t *testing.T) {
	for _, value := range []string{"Adа", "guеst_ada", "Pаypal", "Ελένη Ada"} {
		_, err := players.NameOf(value)
		require.ErrorIs(t, err, players.ErrInvalidName, "%q", value)
	}
}

func TestANameNeverStartsWithTheGuestPrefixInAnyCase(t *testing.T) {
	for _, value := range []string{"guest_ada", "GUEST_ada", "Guest_", "gUeSt_1", "ＧＵＥＳＴ_ada", "gueſt_ada", " guest_ada"} {
		_, err := players.NameOf(value)
		require.ErrorIs(t, err, players.ErrInvalidName, "%q", value)
	}

	for _, value := range []string{"guest", "guestada", "ada_guest_", "guest ada"} {
		_, err := players.NameOf(value)
		require.NoError(t, err, "%q", value)
	}
}

func TestTwoNamesThatDifferOnlyInCaseFoldTheSame(t *testing.T) {
	for _, pair := range [][2]players.Name{
		{"ada_L", "ADA_l"}, {"Émile", "éMILE"}, {"Straße", "STRASSE"}, {"Жанна", "жАННА"}, {"Ａｄａ", "ada"}, {"ΣΊΣΥΦΟΣ", "σίσυφος"},
	} {
		assert.Equal(t, pair[0].Folded(), pair[1].Folded(), "%q and %q", pair[0], pair[1])
	}

	assert.NotEqual(t, players.Name("Ada").Folded(), players.Name("Adá").Folded())
}

func TestAnAccountIDIsAUUIDAndNeverTheNilOne(t *testing.T) {
	account, err := players.AccountIDOf("0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11")
	require.NoError(t, err)
	assert.Equal(t, "0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11", account.String())

	for _, value := range []string{"", "not-a-uuid", "00000000-0000-0000-0000-000000000000"} {
		_, err := players.AccountIDOf(value)
		require.ErrorIs(t, err, players.ErrInvalidAccount, "%q", value)
	}
}
