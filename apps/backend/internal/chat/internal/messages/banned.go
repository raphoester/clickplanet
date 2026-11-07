package messages

import (
	"errors"
	"slices"
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

// The phrases a raid spams, refused at once: the slogans and slurs no conversation makes acceptable, and the symbols.
// Political talk stays (see docs/moderation.md), so a name alone is never listed: "Hitler was a monster" passes.
// chat.service.bannedPhrases adds to them.
var DefaultBannedPhrases = []string{
	"sieg heil", "seig heil", "heil hitler", "hitler did nothing wrong", "gas the jews", "white power", "1488",
	"nigger", "nigga", "faggot",
	"卐", "࿕", "࿗", "ϟϟ",
}

var ErrBanned = errors.New("a banned phrase")

// Banned is a list of phrases, each matched on a folded text: case, accents, Cyrillic and Greek look-alikes,
// leetspeak, separators and repeated letters fold away. A phrase of letters matches from the start of a word, so
// "nigger" refuses "n1gg3r" and "n i g g e r" and lets "Niger" and "snigger" pass. A phrase of no cased letter (卐, a
// CJK word) matches anywhere.
type Banned struct {
	phrases []phrase
}

type phrase struct {
	listed string
	// From the start of a word.
	letters []rune
	// Anywhere, when letters is nil.
	anywhere string
}

func NewBanned(phrases ...string) Banned {
	var banned Banned
	for _, listed := range phrases {
		if p, ok := phraseOf(listed); ok && !slices.ContainsFunc(banned.phrases, p.same) {
			banned.phrases = append(banned.phrases, p)
		}
	}
	return banned
}

func phraseOf(listed string) (phrase, bool) {
	listed = strings.TrimSpace(listed)
	if !slices.ContainsFunc([]rune(norm.NFKD.String(listed)), cased) {
		anywhere := symbolsOf(listed)
		return phrase{listed: listed, anywhere: anywhere}, anywhere != ""
	}
	var letters []rune
	for _, l := range lettersOf(listed) {
		letters = append(letters, l.r)
	}
	return phrase{listed: listed, letters: letters}, len(letters) > 0
}

func (p phrase) same(other phrase) bool {
	return slices.Equal(p.letters, other.letters) && p.anywhere == other.anywhere
}

func cased(r rune) bool {
	return unicode.IsUpper(r) || unicode.IsLower(r) || unicode.IsTitle(r) || unicode.IsDigit(r)
}

// The phrase the text says, as it was listed.
func (b Banned) Said(text string) (string, bool) {
	if len(b.phrases) == 0 {
		return "", false
	}
	letters, symbols := lettersOf(text), symbolsOf(text)
	for _, p := range b.phrases {
		said := p.in(letters)
		if p.letters == nil {
			said = strings.Contains(symbols, p.anywhere)
		}
		if said {
			return p.listed, true
		}
	}
	return "", false
}

// Each letter of the phrase takes one or more of itself, so "niiiggger" says "nigger" and "niger" does not: a run of
// states, one per letter matched, moved by each letter of the text, and begun at each start of a word.
func (p phrase) in(text []letter) bool {
	if len(p.letters) == 0 {
		return false
	}
	active, next := make([]bool, len(p.letters)+1), make([]bool, len(p.letters)+1)
	for _, t := range text {
		clear(next)
		if t.start && alike(p.letters[0], t.r) {
			next[1] = true
		}
		for j := 1; j < len(p.letters); j++ {
			if !active[j] {
				continue
			}
			if alike(p.letters[j-1], t.r) {
				next[j] = true
			}
			if alike(p.letters[j], t.r) {
				next[j+1] = true
			}
		}
		if next[len(p.letters)] {
			return true
		}
		active, next = next, active
	}
	return false
}

// A private-use rune for what reads as an i or an l: 1, | and !.
const iOrL = ''

func alike(listed, said rune) bool {
	switch {
	case listed == said:
		return true
	case listed == iOrL:
		return said == 'i' || said == 'l'
	case said == iOrL:
		return listed == 'i' || listed == 'l'
	default:
		return false
	}
}

type letter struct {
	r rune
	// The first of a word: after a space, a punctuation mark or a symbol, or a letter after a digit or a leet sign.
	start bool
}

type class int

const (
	separator class = iota
	// A letter of the text, as it was written.
	written
	// A digit or a sign that stands for one: 0, 3, @, $.
	leet
)

// Every letter, digit and leet sign of the text, folded, with no separator.
func lettersOf(text string) []letter {
	runes := folded(text)
	letters, before := make([]letter, 0, len(runes)), separator
	for _, r := range runes {
		stands, isLeet := leetspeak[r]
		var now class
		switch {
		case isLeet:
			now, r = leet, stands
		case unicode.IsLetter(r), unicode.IsDigit(r):
			now = written
			if unicode.IsDigit(r) {
				now = leet
			}
		default:
			before = separator
			continue
		}
		start := before == separator || now == written && before == leet
		letters = append(letters, letter{r: r, start: start})
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
