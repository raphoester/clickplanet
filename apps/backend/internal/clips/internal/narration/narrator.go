package narration

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/posts"
)

type Writer interface {
	Write(ctx context.Context, model Model, request Request) (Script, posts.Spend, error)
}

// What the editor was told of one story, alone, as JSON.
type Facts func(findings posts.Findings, story posts.Story) (string, error)

// The war report's writer as the clips job asks it: one tone, one model, the story's facts as the editor read them,
// the findings' day, and the episodes its posts said before.
type Narrator struct {
	writer Writer
	facts  Facts
	tone   Tone
	model  Model
}

func NewNarrator(writer Writer, facts Facts, tone Tone, model Model) Narrator {
	return Narrator{writer: writer, facts: facts, tone: tone, model: model}
}

// A script that cannot be voiced is an error, its cost kept.
func (n Narrator) Narrate(ctx context.Context, findings posts.Findings, story posts.Story, told []posts.Telling) (Script, posts.Spend, error) {
	facts, err := n.facts(findings, story)
	if err != nil {
		return Script{}, posts.Spend{}, fmt.Errorf("failed to read what the editor was told of %s: %w", story.Names(), err)
	}
	request := Request{Story: story, Facts: facts, Tone: n.tone, Day: findings.Day, Episodes: EpisodesOf(told)}
	written, spend, err := n.writer.Write(ctx, n.model, request)
	if err != nil {
		return Script{}, spend, err //nolint:wrapcheck // the writer already says what failed.
	}
	checked, err := posts.ScriptOf(written.Title, written.Lines)
	if err != nil {
		return Script{}, spend, err //nolint:wrapcheck // the domain already says what is wrong.
	}
	return Script{Script: checked, Why: written.Why}, spend, nil
}

// The last episodes are enough to go on from: about 600 words.
const episodesKept = 10

// What the voice said in the last war reports posted, oldest first.
func EpisodesOf(told []posts.Telling) []string {
	voiced := slices.DeleteFunc(slices.Clone(told), func(telling posts.Telling) bool { return len(telling.Said) == 0 })
	slices.SortStableFunc(voiced, func(a, b posts.Telling) int { return a.At.Compare(b.At) })
	voiced = voiced[max(len(voiced)-episodesKept, 0):]

	episodes := make([]string, 0, len(voiced))
	for _, telling := range voiced {
		episodes = append(episodes, strings.Join(telling.Said, " "))
	}
	return episodes
}
