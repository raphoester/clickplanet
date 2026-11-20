// Package bench_usecase makes a round of the war report's bench: the editor chooses the moment's stories, as it would
// for a run, and the writer writes each in each tone with each model.
package bench_usecase

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/narration"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/posts"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/posts/usecases/choose_clips_usecase"
)

type Findings interface {
	Findings(ctx context.Context) (posts.Findings, error)
}

// What the editor was told of one story, alone, as JSON.
type Facts func(findings posts.Findings, story posts.Story) (string, error)

type Writer interface {
	Write(ctx context.Context, model narration.Model, request narration.Request) (narration.Script, posts.Spend, error)
}

// Stories: how many the editor chooses. Day: the season's day the scripts say, 0 for none.
type Round struct {
	Stories int
	Tones   []narration.Tone
	Models  []narration.Model
	Day     int
}

type Executor interface {
	Execute(ctx context.Context) (narration.Bench, error)
}

func New(findings Findings, editor choose_clips_usecase.Editor, facts Facts, writer Writer, round Round) *UseCase {
	return &UseCase{findings: findings, editor: editor, facts: facts, writer: writer, round: round}
}

type UseCase struct {
	findings Findings
	editor   choose_clips_usecase.Editor
	facts    Facts
	writer   Writer
	round    Round
}

var _ Executor = (*UseCase)(nil)

// A script that fails is an entry with why, and the round goes on: one refusal leaves the others to compare.
func (u *UseCase) Execute(ctx context.Context) (narration.Bench, error) {
	found, err := u.findings.Findings(ctx)
	if err != nil {
		return narration.Bench{}, fmt.Errorf("failed to read the findings: %w", err)
	}
	// A day won is told in the generator's words, never a writer's: the editor fills every slot with its own.
	found.Seasonal = nil

	choice := choose_clips_usecase.Edit(ctx, u.editor, found, u.round.Stories, nil, found.Until, posts.LabSlot{})
	bench := narration.Bench{At: found.Until, Editorial: choice.Editorial}
	for _, chosen := range choice.Chosen {
		facts, err := u.facts(found, chosen.Story)
		if err != nil {
			return narration.Bench{}, fmt.Errorf("failed to read what the editor was told of %s: %w", chosen.Story.Names(), err)
		}
		bench.Stories = append(bench.Stories, chosen.Story)
		for _, tone := range u.round.Tones {
			for _, model := range u.round.Models {
				request := narration.Request{Story: chosen.Story, Facts: facts, Tone: tone, Day: u.round.Day}
				script, spend, err := u.writer.Write(ctx, model, request)
				entry := narration.Entry{Story: len(bench.Stories), Tone: tone.Name, Model: model, Script: script, Spend: spend}
				if err != nil {
					entry.Failed = err.Error()
				}
				bench.Entries = append(bench.Entries, entry)
			}
		}
	}
	return bench, nil
}
