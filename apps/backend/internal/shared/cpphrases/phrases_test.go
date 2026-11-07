package cpphrases_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpphrases"
)

func said(match cpphrases.Match, text string, phrases ...string) (string, bool) {
	return cpphrases.New(match, phrases...).Said(text)
}

func TestEverySpellingFoldsToThePhrase(t *testing.T) {
	for _, text := range []string{
		"hitler", "HITLER", "Hítler", "h1tl3r", "h|tler", "hi tler", "h.i.t.l.e.r", "hhhiiitttlller", "ｈｉｔｌｅｒ",
		"𝓱𝓲𝓽𝓵𝓮𝓻", "ʜɪᴛʟᴇʀ", "ⓗⓘⓣⓛⓔⓡ", "🇭🇮🇹🇱🇪🇷", "һіtlеr", // Cyrillic һ, і and е
	} {
		for _, match := range []cpphrases.Match{cpphrases.InWords, cpphrases.Anywhere} {
			phrase, ok := said(match, text, "hitler")
			assert.True(t, ok, "%q", text)
			assert.Equal(t, "hitler", phrase, "%q", text)
		}
	}
}

func TestEachLetterOfAPhraseTakesOneOrMoreOfItself(t *testing.T) {
	_, ok := said(cpphrases.Anywhere, "Niger", "nigger")
	assert.False(t, ok, "the phrase's double g needs two")

	_, ok = said(cpphrases.Anywhere, "niiiggggerrr", "nigger")
	assert.True(t, ok)
}

func TestFromAWordsStartAWordThatEndsInThePhrasePasses(t *testing.T) {
	for text, found := range map[string]bool{
		"snigger": false, "a snigger": false, "s nigger": true, "!nigger": true, "1nigger": true, "niggers": true,
	} {
		_, ok := said(cpphrases.InWords, text, "nigger")
		assert.Equal(t, found, ok, "%q", text)
	}
}

func TestInWordsAPhraseOfManyWordsOrADigitEndsWhereAWordDoes(t *testing.T) {
	for text, found := range map[string]bool{
		"white power": true, "WHITE POWER!!!": true, "white power, brother": true, "white powerrr": true,
		"whitepower": true, "white power88": true, "white power lol": true,
		"the white powers of Europe": false, "white powerful": false, "white powerups": false,
	} {
		_, ok := said(cpphrases.InWords, text, "white power")
		assert.Equal(t, found, ok, "%q", text)
	}

	for text, found := range map[string]bool{
		"1488": true, "14 88": true, "1488!": true, "14888": true, "1488th": true, "I have 1488 tiles": true,
		"14880 tiles": false, "148800": false,
	} {
		_, ok := said(cpphrases.InWords, text, "1488")
		assert.Equal(t, found, ok, "%q", text)
	}
}

func TestInWordsANewWordEndsAPhraseThoughItBeginsWithItsLastLetter(t *testing.T) {
	_, ok := said(cpphrases.InWords, "gg everyone, sieg heil lol", "sieg heil")
	assert.True(t, ok)
}

func TestInWordsAPhraseOfOneWordOfLettersTakesAnyEnding(t *testing.T) {
	for _, text := range []string{"niggers", "niggerz", "faggots"} {
		_, ok := said(cpphrases.InWords, text, "nigger", "faggot")
		assert.True(t, ok, "%q: a slur takes its endings", text)
	}
}

func TestAnywhereAPhraseOfManyWordsMayGoOn(t *testing.T) {
	_, ok := said(cpphrases.Anywhere, "WhitePowerful", "white power")
	assert.True(t, ok, "a name has no word to end")
}

func TestAnywhereThePhraseIsFoundInsideAWord(t *testing.T) {
	for _, text := range []string{"xXHitlerXx", "Shitler", "MrHitler88", "the_hitler_fan"} {
		_, ok := said(cpphrases.Anywhere, text, "hitler")
		assert.True(t, ok, "%q", text)

		_, ok = said(cpphrases.InWords, text, "hitler")
		assert.Equal(t, text == "the_hitler_fan", ok, "%q: only a word that starts with it", text)
	}
}

func TestASymbolIsFoundAnywhereWhateverTheMatch(t *testing.T) {
	for _, match := range []cpphrases.Match{cpphrases.InWords, cpphrases.Anywhere} {
		_, ok := said(match, "ok卐ok", "卐")
		assert.True(t, ok)
	}
}

func TestAPhraseThatFoldsToNothingIsLeftOut(t *testing.T) {
	_, ok := said(cpphrases.Anywhere, "hello planet", "", "   ", "!!!", "...")
	assert.False(t, ok, "an empty phrase would match every text")
}

func TestNoListFindsNothing(t *testing.T) {
	_, ok := said(cpphrases.Anywhere, "hitler")
	assert.False(t, ok)
}

func TestThePhraseIsAnsweredAsItWasListed(t *testing.T) {
	phrase, ok := said(cpphrases.InWords, "S1EG H3IL!!", "  Sieg Heil ")

	assert.True(t, ok)
	assert.Equal(t, "Sieg Heil", phrase)
}

func TestADigitOfATextIsALetterAndADigitOfAPhraseOnlyItself(t *testing.T) {
	for _, match := range []cpphrases.Match{cpphrases.InWords, cpphrases.Anywhere} {
		_, ok := said(match, "n1gg3r", "nigger")
		assert.True(t, ok, "a text's 1 and 3 read as an i and an e")

		_, ok = said(match, "14 88", "1488")
		assert.True(t, ok, "a phrase's digits match the text's")

		for _, text := range []string{"BraveKoala88", "la 88e minute", "Ila88"} {
			_, ok = said(match, text, "1488")
			assert.False(t, ok, "%q: a letter is never a phrase's digit", text)
		}
	}
}
