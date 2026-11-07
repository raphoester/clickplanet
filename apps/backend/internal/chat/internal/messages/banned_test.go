package messages_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
)

var banned = messages.NewBanned(messages.DefaultBannedPhrases...)

func TestABannedPhraseIsFoundHoweverItIsWritten(t *testing.T) {
	for text, phrase := range map[string]string{
		"nigger":                     "nigger",
		"NIGGER":                     "nigger",
		"Nígger":                     "nigger",
		"nіgger":                     "nigger", // a Cyrillic і
		"ηιggεr":                     "nigger", // Greek
		"n1gg3r":                     "nigger",
		"n!99er":                     "nigger",
		"n|gger":                     "nigger",
		"n.i.g.g.e.r":                "nigger",
		"n i g g e r":                "nigger",
		"n-i_g*g~e r":                "nigger",
		"niiiiggggerrrr":             "nigger",
		"ｎｉｇｇｅｒ":                     "nigger", // fullwidth
		"𝓷𝓲𝓰𝓰𝓮𝓻":                     "nigger", // mathematical script
		"ɴɪɢɢᴇʀ":                     "nigger", // small capitals
		"ⓝⓘⓖⓖⓔⓡ":                     "nigger", // circled
		"🇳🇮🇬🇬🇪🇷":                     "nigger", // regional indicators
		"n\u200bigger":               "nigger", // a zero-width space
		"n😀i😀g😀g😀e😀r":                "nigger",
		"niggers everywhere":         "nigger",
		"!nigger":                    "nigger",
		"@nigger":                    "nigger",
		"gg everyone, sieg heil lol": "sieg heil",
		"Sieg Heil!!!":               "sieg heil",
		"SIEG-HEIL":                  "sieg heil",
		"siegheil":                   "sieg heil",
		"s1eg h3il":                  "sieg heil",
		"ЅІЕG НЕІL":                  "sieg heil", // Cyrillic capitals
		"seig heil":                  "seig heil",
		"heil hitler":                "heil hitler",
		"he1l h1tler":                "heil hitler",
		"hei1 hit1er":                "heil hitler", // a 1 for an l
		"14 88":                      "1488",
		"WHITE POWER":                "white power",
		"f4gg0t":                     "faggot",
	} {
		said, ok := banned.Said(text)
		require.True(t, ok, "%q", text)
		assert.Equal(t, phrase, said, "%q", text)
	}
}

func TestAWordThatOnlyLooksLikeABannedPhrasePasses(t *testing.T) {
	for _, text := range []string{
		"Niger", "I live in Niger", "Nigeria attacks", "nigerian flag",
		"snigger", "sniggering", "niggle", "a niggling doubt",
		"hell yeah", "go to hell hitler", "Hitler was a monster", "the nazis lost the war",
		"white powder", "whitepaper", "a fagot is a bassoon",
		"I have 88 tiles", "1489", "à la 88e minute", "gg France 🇫🇷🔥", "Привет", "東京タワー",
	} {
		said, ok := banned.Said(text)
		assert.False(t, ok, "%q said %q", text, said)
	}
}

func TestASymbolIsFoundAnywhere(t *testing.T) {
	for _, text := range []string{"卐", "ok卐ok", "a 卐 b", "࿕", "ϟϟ"} {
		_, ok := banned.Said(text)
		assert.True(t, ok, "%q", text)
	}
}

func TestAPhraseOfNoLetterThatFoldsToNothingIsNeverListed(t *testing.T) {
	nothing := messages.NewBanned("", "   ", "!!!", "...")

	_, ok := nothing.Said("hello planet")
	assert.False(t, ok, "an empty phrase would refuse every text")
}

func TestNoListFindsNothing(t *testing.T) {
	_, ok := messages.NewBanned().Said("nigger")
	assert.False(t, ok)
}

func TestALimitWithTheListRefusesABannedTextAsAnInvalidMessage(t *testing.T) {
	limits := messages.NewLimits(0).WithBanned(banned)

	_, err := limits.Text("  s.i.e.g  h.e.i.l  ")

	require.ErrorIs(t, err, messages.ErrInvalidMessage, "the sender is told what any other refusal tells")
	require.ErrorIs(t, err, messages.ErrBanned)
	assert.Contains(t, err.Error(), `"sieg heil"`, "the log names the phrase")
}

func TestALimitWithTheListLetsAnyOtherTextGo(t *testing.T) {
	limits := messages.NewLimits(0).WithBanned(banned)

	text, err := limits.Text("gg Niger, see you on the map")

	require.NoError(t, err)
	assert.Equal(t, "gg Niger, see you on the map", text)
}

func TestAPhraseTheConfigAddsIsRefusedToo(t *testing.T) {
	limits := messages.NewLimits(0).WithBanned(messages.NewBanned("pony raid"))

	_, err := limits.Text("P0NY RA1D")
	require.ErrorIs(t, err, messages.ErrBanned)

	_, err = limits.Text("ponies")
	require.NoError(t, err)
}

func TestAGifIsNotReadForPhrases(t *testing.T) {
	limits := messages.NewLimits(0).WithBanned(messages.NewBanned("klipy"))

	_, gif, err := limits.Body("", messages.Gif{URL: "https://static.klipy.com/ii/935d7ab9/14/af/SE72470w.webp", Width: 220, Height: 124})

	require.NoError(t, err)
	assert.False(t, gif.None())
}
