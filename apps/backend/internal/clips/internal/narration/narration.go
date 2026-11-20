// Package narration is what a narrated clip's voice says (docs/video-lab.md, "The war report's bench"): a script
// written to a tone by a model, from one story of a run, and how it is measured before anyone listens to it.
package narration

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"unicode"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/posts"
)

// The name of a tone, as its brief's file is named: "correspondent".
type ToneName string

// How a script sounds: a brief the writer gets beside its instructions.
type Tone struct {
	Name  ToneName
	Brief string
}

// A model that writes scripts.
type Model string

const (
	Haiku  Model = "claude-haiku-5-5"
	Sonnet Model = "claude-sonnet-5-5"
	Opus   Model = "claude-opus-5-5"
)

var ErrUnknownModel = errors.New("no such writer")

// "haiku", "sonnet" or "opus", or a model's own id.
func ModelOf(name string) (Model, error) {
	known := map[string]Model{"haiku": Haiku, "sonnet": Sonnet, "opus": Opus}
	name = strings.ToLower(strings.TrimSpace(name))
	if model, ok := known[name]; ok {
		return model, nil
	}
	for _, model := range known {
		if string(model) == name {
			return model, nil
		}
	}
	return "", fmt.Errorf("%w: %q is not haiku, sonnet or opus", ErrUnknownModel, name)
}

// What a writer is asked: one story the editor chose, what it was told of it, the tone, the season's day (0 when
// unknown), and the scripts of the episodes before it, oldest first, so the series goes on from where it was.
type Request struct {
	Story    posts.Story
	Facts    string
	Tone     Tone
	Day      int
	Episodes []string
}

// What the voice says, a sentence a line in the order said, and the post's title. Why is the writer's own note, never
// said.
type Script struct {
	Title string
	Lines []string
	Why   string
}

// A narrator's pace: 156 words a minute.
const wordsASecond = 2.6

func (s Script) Said() string {
	return strings.Join(s.Lines, " ")
}

func (s Script) Words() int {
	return len(strings.Fields(s.Said()))
}

// How long the voice takes to say it, at a narrator's pace.
func (s Script) Seconds() float64 {
	return float64(s.Words()) / wordsASecond
}

// The turns that give a machine away, as they are found in it, and the signs a voice reads badly. The bench shows
// them and refuses nothing: whether a script sounds like a person is the owner's ear to say.
func (s Script) Tells() []string {
	said := strings.ToLower(s.Said())
	var found []string
	for _, tell := range machineTurns {
		if strings.Contains(said, tell) {
			found = append(found, tell)
		}
	}
	if strings.ContainsAny(said, "—–") {
		found = append(found, "a dash")
	}
	if strings.ContainsFunc(said, unicode.IsDigit) {
		found = append(found, "a digit")
	}
	if strings.Count(said, "!") > 1 {
		found = append(found, "exclamation marks")
	}
	return found
}

// Lower case, as they are said. The writer's instructions name them too.
var machineTurns = slices.Sorted(slices.Values([]string{
	"turn of events", "little did", "buckle up", "let that sink in", "it's not just", "it is not just",
	"here's the thing", "only time will tell", "the stage is set", "in a world where", "epic", "showdown",
	"unfold", "testament", "tapestry", "delve", "embark", "against all odds", "game-changer", "game changer",
	"make no mistake", "the question remains", "one thing is clear", "stay tuned", "folks",
}))
