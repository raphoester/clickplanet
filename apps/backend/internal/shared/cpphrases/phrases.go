// Package cpphrases finds a listed phrase in a text however it is spelled: case, accents, Cyrillic and Greek
// look-alikes, the small capitals of the "fancy text" generators, the regional indicators a flag is made of,
// leetspeak, separators and repeated letters all fold away, on the phrase and on the text alike. It knows no list:
// each context keeps its own, and says where a phrase may begin.
package cpphrases

import (
	"slices"
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

// Where a phrase of letters may begin and end. A phrase with no cased letter or digit (卐, a CJK word) matches anywhere
// whatever the match: it has no word to start.
type Match uint8

const (
	// A sentence's. A phrase begins where a word does: after a space, a punctuation mark or a symbol, or a letter after
	// a digit or a leet sign. So "nigger" refuses "n i g g e r" and "!nigger", and lets "snigger" pass. A phrase of one
	// word of letters takes any ending, as a slur does ("niggers"); any other, of many words or with a digit, is whole
	// and ends where a word does, so "white power" lets "the white powers of Europe" pass and "1488" lets "14880".
	InWords Match = iota
	// Anywhere in the text, and a phrase may go on: "hitler" refuses "xXHitlerXx". A name's, which is one word.
	Anywhere
)

// Each letter of a phrase takes one or more of itself, so "niiiggger" says "nigger" and "Niger" does not: never a
// collapse of the text's repeated letters, which would make the two one.
type List struct {
	phrases []phrase
	match   Match
}

type phrase struct {
	listed string
	// From the start of a word.
	letters []rune
	// Of many words, or with a digit: ends where a word does, InWords.
	whole bool
	// Anywhere, when letters is nil.
	anywhere string
}

// A phrase that folds to nothing ("", "!!!") is left out: it would match every text.
func New(match Match, phrases ...string) List {
	list := List{match: match}
	for _, listed := range phrases {
		if p, ok := phraseOf(listed); ok && !slices.ContainsFunc(list.phrases, p.same) {
			list.phrases = append(list.phrases, p)
		}
	}
	return list
}

func phraseOf(listed string) (phrase, bool) {
	listed = strings.TrimSpace(listed)
	if !slices.ContainsFunc([]rune(norm.NFKD.String(listed)), cased) {
		anywhere := symbolsOf(listed)
		return phrase{listed: listed, anywhere: anywhere}, anywhere != ""
	}
	var letters []rune
	words := 0
	for _, l := range lettersOf(listed, asListed) {
		letters = append(letters, l.r)
		if l.start {
			words++
		}
	}
	whole := words > 1 || slices.ContainsFunc(letters, unicode.IsDigit)
	return phrase{listed: listed, letters: letters, whole: whole}, len(letters) > 0
}

func (p phrase) same(other phrase) bool {
	return slices.Equal(p.letters, other.letters) && p.whole == other.whole && p.anywhere == other.anywhere
}

func cased(r rune) bool {
	return unicode.IsUpper(r) || unicode.IsLower(r) || unicode.IsTitle(r) || unicode.IsDigit(r)
}

// The phrase the text says, as it was listed.
func (l List) Said(text string) (string, bool) {
	if len(l.phrases) == 0 {
		return "", false
	}
	letters, symbols := lettersOf(text, asLeet), symbolsOf(text)
	for _, p := range l.phrases {
		said := p.in(letters, l.match)
		if p.letters == nil {
			said = strings.Contains(symbols, p.anywhere)
		}
		if said {
			return p.listed, true
		}
	}
	return "", false
}

// A run of states, one per letter matched, moved by each letter of the text, and begun where the match lets a phrase
// begin. A whole phrase, InWords, is said once the word it ends in ends: at the text's end, or at a letter that does
// not go on with it.
func (p phrase) in(text []letter, match Match) bool {
	if len(p.letters) == 0 {
		return false
	}
	last, whole := len(p.letters), p.whole && match == InWords
	active, next := make([]bool, last+1), make([]bool, last+1)
	for _, t := range text {
		// A new word ends it, even one that begins with its last letter ("sieg heil lol"); a repeat of that letter does
		// not ("white powerrr").
		if active[last] && (t.start || !alike(p.letters[last-1], t) && !p.goesOn(t)) {
			return true
		}
		clear(next)
		if (t.start || match == Anywhere) && alike(p.letters[0], t) {
			next[1] = true
		}
		for j := 1; j <= last; j++ {
			if !active[j] {
				continue
			}
			if alike(p.letters[j-1], t) {
				next[j] = true
			}
			if j < last && alike(p.letters[j], t) {
				next[j+1] = true
			}
		}
		if next[last] && !whole {
			return true
		}
		active, next = next, active
	}
	return active[last]
}

// Whether a letter of the text makes one word with the phrase that ended before it: a letter after a letter, a digit
// after a digit, never a new word's start. So "white powerful" and "14880" are other words, and "WHITE POWER!!!" and
// "1488th" say the phrase.
func (p phrase) goesOn(t letter) bool {
	if t.start {
		return false
	}
	if unicode.IsDigit(p.letters[len(p.letters)-1]) {
		return unicode.IsDigit(t.written)
	}
	return unicode.IsLetter(t.written)
}

// A private-use rune for what reads as an i or an l: 1, | and !.
const iOrL = '\uE000'

// A text's digit is a letter or itself, and a phrase's digit only itself: "n1gg3r" says "nigger", and "Koala88" never
// says "1488" (its l is no 1).
func alike(listed rune, said letter) bool {
	switch {
	case listed == said.r, listed == said.written:
		return true
	case said.r == iOrL:
		return listed == 'i' || listed == 'l'
	default:
		return false
	}
}

type letter struct {
	// Folded, and in a text, a digit or a leet sign as the letter it stands for.
	r rune
	// Folded, as written: a digit stays one.
	written rune
	// The first of a word: after a space, a punctuation mark or a symbol, or a letter after a digit or a leet sign.
	start bool
}

// How a digit or a leet sign is read: in a text as the letter it stands for, in a phrase as it was listed.
type reading bool

const (
	asLeet   reading = true
	asListed reading = false
)

type class int

const (
	separator class = iota
	// A letter of the text, as it was written.
	written
	// A digit or a sign that stands for one: 0, 3, @, $.
	leet
)

// Every letter, digit and leet sign of the text, folded, with no separator.
func lettersOf(text string, read reading) []letter {
	runes := folded(text)
	letters, before := make([]letter, 0, len(runes)), separator
	for _, r := range runes {
		stands, isLeet := leetspeak[r]
		var now class
		switch {
		case unicode.IsDigit(r):
			now = leet
		case isLeet && read == asLeet:
			now = leet
		case unicode.IsLetter(r):
			now = written
		default:
			before = separator
			continue
		}
		found := letter{r: r, written: r}
		if isLeet && read == asLeet {
			found.r = stands
		}
		found.start = before == separator || now == written && before == leet
		letters = append(letters, found)
		before = now
	}
	return letters
}

// What the text shows, folded, with no space, punctuation mark or control character: where a symbol is looked for.
func symbolsOf(text string) string {
	var b strings.Builder
	for _, r := range folded(text) {
		if unicode.IsSpace(r) || unicode.IsPunct(r) || unicode.IsControl(r) || unicode.Is(unicode.Cf, r) {
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

// Decomposed (a fullwidth or a mathematical letter becomes its letter), with no accent, in lower case, a look-alike
// as the Latin letter it passes for.
func folded(text string) []rune {
	var runes []rune
	for _, r := range norm.NFKD.String(text) {
		if unicode.Is(unicode.Mn, r) {
			continue
		}
		r = unicode.ToLower(r)
		if latin, ok := lookAlikes[r]; ok {
			runes = append(runes, []rune(latin)...)
			continue
		}
		if r >= regionalA && r <= regionalZ {
			r = 'a' + r - regionalA
		}
		runes = append(runes, r)
	}
	return runes
}

// The letters a flag is made of: 🇳🇮 is two of them.
const (
	regionalA = '\U0001F1E6'
	regionalZ = '\U0001F1FF'
)

var leetspeak = map[rune]rune{
	'0': 'o', '1': iOrL, '3': 'e', '4': 'a', '5': 's', '7': 't', '8': 'b', '9': 'g',
	'@': 'a', '$': 's', '€': 'e', '!': iOrL, '|': iOrL, '¡': iOrL,
}

// What NFKD does not decompose: the Cyrillic and Greek letters that pass for Latin ones, the Latin letters with a
// stroke, and the small capitals of the "fancy text" generators.
var lookAlikes = map[rune]string{
	// Cyrillic.
	'а': "a", 'в': "b", 'ь': "b", 'с': "c", 'ԁ': "d", 'е': "e", 'һ': "h", 'н': "h", 'і': "i", 'ӏ': "l", 'ј': "j",
	'к': "k", 'м': "m", 'п': "n", 'о': "o", 'р': "p", 'ԛ': "q", 'ѕ': "s", 'т': "t", 'у': "y", 'ү': "y", 'х': "x",
	'ԝ': "w",
	// Greek.
	'α': "a", 'β': "b", 'ϲ': "c", 'ε': "e", 'η': "n", 'ι': "i", 'ϳ': "j", 'κ': "k", 'μ': "u", 'ν': "v", 'ο': "o",
	'π': "n", 'ρ': "p", 'τ': "t", 'υ': "u", 'χ': "x", 'γ': "y", 'ω': "w",
	// Latin.
	'ı': "i", 'ł': "l", 'ø': "o", 'đ': "d", 'ħ': "h", 'ŧ': "t", 'ß': "ss", 'æ': "ae", 'œ': "oe", 'ɡ': "g", 'ɑ': "a",
	'ɩ': "i",
	// Small capitals.
	'ᴀ': "a", 'ʙ': "b", 'ᴄ': "c", 'ᴅ': "d", 'ᴇ': "e", 'ғ': "f", 'ɢ': "g", 'ʜ': "h", 'ɪ': "i", 'ᴊ': "j", 'ᴋ': "k",
	'ʟ': "l", 'ᴍ': "m", 'ɴ': "n", 'ᴏ': "o", 'ᴘ': "p", 'ʀ': "r", 'ꜱ': "s", 'ᴛ': "t", 'ᴜ': "u", 'ᴠ': "v", 'ᴡ': "w",
	'ʏ': "y", 'ᴢ': "z",
}
