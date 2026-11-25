package posts

import (
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"
)

// What a war report's voice says (docs/video-lab.md), a sentence a line in the order said, and the post's title, which
// the caption opens with in place of a teaser.
type Script struct {
	Title string
	Lines []string
}

// The voice says about 3.3 words a second from 0.4 s in, and the clip ends 2.9 s after its last word, 30 s at most.
const scriptWords = 80

var ErrInvalidScript = errors.New("the script cannot be voiced")

// The title in capitals and short enough for YouTube's with the brand, each line on one line, and no more words than
// a clip of 30 s says.
func ScriptOf(title string, lines []string) (Script, error) {
	script := Script{Title: headlineOf(oneLine(title))}
	if script.Title == "" || utf8.RuneCountInString(script.Title) > teaserRunes {
		return Script{}, fmt.Errorf("%w: the title %q is empty or over %d letters", ErrInvalidScript, script.Title, teaserRunes)
	}
	for _, line := range lines {
		if said := oneLine(line); said != "" {
			script.Lines = append(script.Lines, said)
		}
	}
	if len(script.Lines) == 0 {
		return Script{}, fmt.Errorf("%w: it says nothing", ErrInvalidScript)
	}
	if words := len(strings.Fields(strings.Join(script.Lines, " "))); words > scriptWords {
		return Script{}, fmt.Errorf("%w: %d words are past the %d a clip of 30 s says", ErrInvalidScript, words, scriptWords)
	}
	return script, nil
}

func (s Script) Voiced() bool {
	return len(s.Lines) > 0
}
