package messages

import (
	"errors"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpphrases"
)

// The phrases a raid spams, refused at once: the slogans and slurs no conversation makes acceptable, and the symbols.
// Political talk stays (see docs/moderation.md), so a name alone is never listed: "Hitler was a monster" passes. So do
// the insults the policy keeps, "nigga" (the open-a form players call each other) and "faggot"; the hard-r form is
// listed. chat.service.bannedPhrases adds to them.
var DefaultBannedPhrases = []string{
	"sieg heil", "seig heil", "heil hitler", "hitler did nothing wrong", "gas the jews", "white power", "1488",
	"nigger",
	"卐", "࿕", "࿗", "ϟϟ",
}

var ErrBanned = errors.New("a banned phrase")

// The phrases a message may not say, folded however they are spelled (cpphrases).
type Banned = cpphrases.List

// A message is sentences: a phrase of letters matches from the start of a word, so "nigger" refuses "n1gg3r" and lets
// "Niger" and "snigger" pass, and one of many words or a number ends where a word does, so "white power" lets "the
// white powers of Europe" pass.
func NewBanned(phrases ...string) Banned {
	return cpphrases.New(cpphrases.InWords, phrases...)
}
