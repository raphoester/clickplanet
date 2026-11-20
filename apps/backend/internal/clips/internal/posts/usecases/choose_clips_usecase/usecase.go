package choose_clips_usecase

import (
	"context"
	"fmt"
	"time"

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

type Room struct {
	PerRun int
	Plan   posts.Plan
	// The experiments on: each run rolls whether one of its picks is theirs (docs/video-lab.md).
	Lab posts.Lab
}

type Executor interface {
	Execute(ctx context.Context) (posts.Choice, error)
}

func New(findings Findings, history History, rooms Rooms, clock cptime.Clock, room Room, editor Editor, dice posts.Dice) *UseCase {
	return &UseCase{findings: findings, history: history, rooms: rooms, clock: clock, room: room, editor: editor, dice: dice}
}

type UseCase struct {
	findings Findings
	history  History
	rooms    Rooms
	clock    cptime.Clock
	room     Room
	editor   Editor
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

	return Edit(ctx, u.editor, findings, count, told, now, u.room.Lab.Slot(count, u.dice)), nil
}

// No call when nothing is left to decide; an editor that cannot answer leaves the run its seasonal stories. lab: the
// pick the lab's experiment fills, none for a run of the control alone.
func Edit(ctx context.Context, editor Editor, findings posts.Findings, count int, told []posts.Telling, now time.Time, lab posts.LabSlot) posts.Choice {
	offer := findings.Offer(count, told, now, lab)
	if !offer.Open() {
		return findings.Edited(offer, posts.Edition{}, told, now)
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
	return choice
}
