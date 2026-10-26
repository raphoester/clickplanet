package choose_clips_usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/posts"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Shortlists interface {
	Shortlist(ctx context.Context) (posts.Shortlist, error)
}

type History interface {
	// Every post with a story key that goes out at since or later, and maybe some before.
	Told(ctx context.Context, since time.Time) ([]posts.Telling, error)
}

type Feeds interface {
	// Every post, by hand or not, that goes out at since or later on each platform, and maybe some before.
	Posted(ctx context.Context, since time.Time) (posts.Feed, error)
}

// Ranks the stories offered and writes their headlines. The spend is what the call cost, whatever its error, with its
// model named whenever a request was sent, and zero when none was (no key); an error is any answer the run cannot
// use, which the run then makes by the rules.
type Editor interface {
	Edit(ctx context.Context, offer posts.Offer) (posts.Edition, posts.Spend, error)
}

// How many clips a run makes: perRun at most, within each platform's day.
type Room struct {
	PerRun    int
	Budget    posts.Budget
	Platforms []posts.Platform
}

type Executor interface {
	Execute(ctx context.Context) (posts.Choice, error)
}

// dice: the coins that put each story chosen in an arm of every experiment. editor: ranks the stories the rules let
// through and writes their headlines; the rules make the run when it cannot.
func New(
	shortlists Shortlists, history History, feeds Feeds, clock cptime.Clock, room Room, experiments posts.Experiments,
	dice posts.Dice, editor Editor,
) *UseCase {
	return &UseCase{
		shortlists: shortlists, history: history, feeds: feeds, clock: clock, room: room, experiments: experiments, dice: dice,
		editor: editor,
	}
}

type UseCase struct {
	shortlists  Shortlists
	history     History
	feeds       Feeds
	clock       cptime.Clock
	room        Room
	experiments posts.Experiments
	dice        posts.Dice
	editor      Editor
}

var _ Executor = (*UseCase)(nil)

func (u *UseCase) Execute(ctx context.Context) (posts.Choice, error) {
	shortlist, err := u.shortlists.Shortlist(ctx)
	if err != nil {
		return posts.Choice{}, fmt.Errorf("failed to read the stories worth a clip: %w", err)
	}

	now := u.clock.Now()
	told, err := u.history.Told(ctx, posts.ToldSince(now))
	if err != nil {
		return posts.Choice{}, fmt.Errorf("failed to read the stories already told: %w", err)
	}
	feed, err := u.feeds.Posted(ctx, posts.PostedSince(now))
	if err != nil {
		return posts.Choice{}, fmt.Errorf("failed to read what each platform holds: %w", err)
	}
	count := u.room.Budget.Clips(u.room.PerRun, u.room.Platforms, feed, now)

	choice, headlines := Edit(ctx, u.editor, shortlist, count, told, now)
	choice.Assigned = u.experiments.AssignEach(choice.Chosen, headlines, u.dice)
	return choice, nil
}

// The run as the editor tells it within the rules, with each headline it changed by pick; the rules' choice when
// nothing is left to decide (no call), and when the editor could not answer or answered what does not hold (a fall
// back, and why, in the choice's Editorial).
func Edit(
	ctx context.Context, editor Editor, shortlist posts.Shortlist, count int, told []posts.Telling, now time.Time,
) (posts.Choice, map[posts.Pick]string) {
	ruled := shortlist.Best(count, told, now)
	offer := shortlist.Offer(count, told, now)
	if !offer.Open() {
		return ruled, nil
	}

	edition, spend, err := editor.Edit(ctx, offer)
	ruled.Editorial.Asked, ruled.Editorial.Spend = spend.Model != "", spend
	if err != nil {
		ruled.Editorial.FellBack = err.Error()
		return ruled, nil
	}
	ruled.Editorial.Why = edition.Why

	choice, headlines, err := shortlist.Edited(offer, edition, told, now)
	if err != nil {
		ruled.Editorial.FellBack = err.Error()
		return ruled, nil
	}
	choice.Editorial = ruled.Editorial
	return choice, headlines
}
