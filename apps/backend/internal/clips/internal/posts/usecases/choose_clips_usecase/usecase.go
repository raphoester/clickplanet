package choose_clips_usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/narration"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/posts"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Findings interface {
	Findings(ctx context.Context) (posts.Findings, error)
}

type History interface {
	// Every post with a story key that goes out at since or later, and maybe some before.
	Told(ctx context.Context, since time.Time) ([]posts.Telling, error)
}

// The publishing module's calendar.
type Rooms interface {
	// How many more posts each platform takes now, by hand or not.
	Room(ctx context.Context, platforms []posts.Platform) (map[posts.Platform]int, error)
}

// The spend names its model whenever a request was sent; an error is any answer the run cannot read.
type Editor interface {
	Edit(ctx context.Context, offer posts.Offer) (posts.Edition, posts.Spend, error)
}

// Writes what the voice of a narrated experiment's story says, checked; an error is a script that cannot be voiced.
type Narrator interface {
	Narrate(ctx context.Context, findings posts.Findings, story posts.Story, told []posts.Telling) (narration.Script, posts.Spend, error)
}

type Room struct {
	PerRun int
	Plan   posts.Plan
	// The experiments on: each run rolls whether one of its picks is theirs (docs/video-lab.md).
	Lab posts.Lab
}

type Executor interface {
	Execute(ctx context.Context) (posts.Choice, error)
}

func New(findings Findings, history History, rooms Rooms, clock cptime.Clock, room Room, editor Editor, narrator Narrator, dice posts.Dice) *UseCase {
	return &UseCase{findings: findings, history: history, rooms: rooms, clock: clock, room: room, editor: editor, narrator: narrator, dice: dice}
}

type UseCase struct {
	findings Findings
	history  History
	rooms    Rooms
	clock    cptime.Clock
	room     Room
	editor   Editor
	narrator Narrator
	dice     posts.Dice
}

var _ Executor = (*UseCase)(nil)

func (u *UseCase) Execute(ctx context.Context) (posts.Choice, error) {
	findings, err := u.findings.Findings(ctx)
	if err != nil {
		return posts.Choice{}, fmt.Errorf("failed to read the findings: %w", err)
	}

	now := u.clock.Now()
	told, err := u.history.Told(ctx, posts.ToldSince(now))
	if err != nil {
		return posts.Choice{}, fmt.Errorf("failed to read the stories already told: %w", err)
	}
	room, err := u.rooms.Room(ctx, u.room.Plan.Platforms)
	if err != nil {
		return posts.Choice{}, fmt.Errorf("failed to read what each platform has room for: %w", err)
	}
	count := u.room.Plan.Clips(u.room.PerRun, room)

	return EditNarrated(ctx, u.editor, u.narrator, findings, count, told, now, u.room.Lab.Slot(count, u.dice)), nil
}

// No call when nothing is left to decide; an editor that cannot answer leaves the run its seasonal stories. lab: the
// pick the lab's experiment fills, none for a run of the control alone.
func Edit(ctx context.Context, editor Editor, findings posts.Findings, count int, told []posts.Telling, now time.Time, lab posts.LabSlot) posts.Choice {
	choice, _, _ := edited(ctx, editor, findings, count, told, now, lab)
	return choice
}

// Edit, then the voice of a narrated experiment's story written: a story the writer gives no script leaves its pick to
// the control's next story, as a story that does not hold does.
func EditNarrated(ctx context.Context, editor Editor, narrator Narrator, findings posts.Findings, count int, told []posts.Telling, now time.Time, lab posts.LabSlot) posts.Choice {
	choice, offer, edition := edited(ctx, editor, findings, count, told, now, lab)
	story, ok := choice.LabStory()
	if !ok || !offer.Lab.Experiment.Narrated {
		return choice
	}

	script, spend, err := narrator.Narrate(ctx, findings, story, offer.Told)
	narrated := posts.Editorial{Asked: spend.Model != "", Spend: spend, Why: script.Why}
	if err != nil {
		narrated.Failed = err.Error()
		unvoiced := findings.Edited(offer, edition.WithoutLab(), told, now)
		unvoiced.Editorial = choice.Editorial
		choice = unvoiced.Skip(story, fmt.Sprintf("the lab's %s has no voice: %s", offer.Lab.Experiment.Name, err))
	} else {
		choice = choice.Voiced(script.Script)
	}
	choice.Narration = narrated
	return choice
}

func edited(ctx context.Context, editor Editor, findings posts.Findings, count int, told []posts.Telling, now time.Time, lab posts.LabSlot) (posts.Choice, posts.Offer, posts.Edition) {
	offer := findings.Offer(count, told, now, lab)
	if !offer.Open() {
		return findings.Edited(offer, posts.Edition{}, told, now), offer, posts.Edition{}
	}

	edition, spend, err := editor.Edit(ctx, offer)
	editorial := posts.Editorial{Asked: spend.Model != "", Spend: spend}
	if err != nil {
		editorial.Failed = err.Error()
		edition = posts.Edition{}
	} else {
		editorial.Why = edition.Why
	}
	choice := findings.Edited(offer, edition, told, now)
	choice.Editorial = editorial
	return choice, offer, edition
}
