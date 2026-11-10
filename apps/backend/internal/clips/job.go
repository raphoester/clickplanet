package clips

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/posts"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/posts/claude_editor"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/posts/generator_output"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/posts/log_history"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/posts/log_service"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/posts/ssh_appearance_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/posts/ssh_publishing"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/posts/usecases/choose_clips_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/posts/usecases/choose_clips_usecase/log_choose_clips"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/posts/usecases/keep_appearances_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/posts/usecases/keep_appearances_usecase/log_keep_appearances"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/posts/usecases/schedule_clips_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/posts/usecases/schedule_clips_usecase/log_schedule_clips"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/replays"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/replays/file_replay_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/replays/ssh_replay_source"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/replays/usecases/fetch_cast_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/replays/usecases/fetch_cast_usecase/log_fetch_cast"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/replays/usecases/fetch_replay_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/replays/usecases/fetch_replay_usecase/log_fetch_replay"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/replays/usecases/fetch_round_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/replays/usecases/fetch_round_usecase/log_fetch_round"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/replays/usecases/fetch_turn_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/replays/usecases/fetch_turn_usecase/log_fetch_turn"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpanthropic"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type (
	Pick            = posts.Pick
	Story           = posts.Story
	Chosen          = posts.Chosen
	Skipped         = posts.Skipped
	Offer           = posts.Offer
	Findings        = posts.Findings
	Seasonal        = posts.Seasonal
	Assignment      = posts.Assignment
	Choice          = posts.Choice
	Editorial       = posts.Editorial
	Variants        = posts.Variants
	Platform        = posts.Platform
	Report          = posts.Report
	Outcome         = posts.Outcome
	PlatformOutcome = posts.PlatformOutcome
)

type Config struct {
	Replay ReplayConfig

	PerRun int
	// Zero is none: a post goes out as soon as its clip is rendered.
	Review time.Duration
	// Each clip is one post on all of them that have room left in their day.
	Platforms []Platform
	// The editor's. No key is no editor: every run is its seasonal stories alone, and says why.
	Anthropic cpanthropic.Config
}

type ReplayConfig struct {
	// user@host of the machine the backend runs on: where the replay is asked, and each post planned and made.
	SSH  string
	Span time.Duration
	// Zero is now.
	Until time.Time
}

const (
	// The ledger keeps this much in memory: a replay cannot reach further back.
	defaultSpan = 72 * time.Hour

	// A run every 2 hours: 2 a run is 24 a day at most, which the publishing module's calendar brings down.
	defaultPerRun = 2

	// A 22s clip is about 80 MB to upload, straight to the posting service's storage.
	uploadTimeout = 10 * time.Minute
)

var (
	defaultBackoff   = posts.Backoff{time.Minute, 5 * time.Minute, 15 * time.Minute}
	defaultPlatforms = []Platform{posts.TikTok, posts.YouTube}
	// TikTok takes a run's best clip alone: its views per clip fell from about 600 to 15 the day the job went to 10 posts a
	// day there (2026-10-19). How many a day, and when, is the publishing module's calendar.
	bestOnly = []Platform{posts.TikTok}
)

const TikTok = posts.TikTok

// What a call of input tokens in and output out costs the editor's model.
func SpendOf(input, output int64) posts.Spend {
	return claude_editor.SpendOf(claude_editor.DefaultModel, input, 0, 0, output)
}

// A story as the generator's --story takes it, in JSON.
func StoryJSON(story Story) ([]byte, error) {
	return generator_output.StoryJSON(story) //nolint:wrapcheck // a pure delegation: the adapter already named what failed.
}

func PlatformsOf(names ...string) ([]Platform, error) {
	return posts.PlatformsOf(names...) //nolint:wrapcheck // a pure delegation: the domain already named what is wrong.
}

func (c Config) withDefaults() Config {
	if c.Replay.Span == 0 {
		c.Replay.Span = defaultSpan
	}
	if c.PerRun <= 0 {
		c.PerRun = defaultPerRun
	}
	c.Review = max(c.Review, 0)
	if len(c.Platforms) == 0 {
		c.Platforms = defaultPlatforms
	}
	return c
}

// turn, round, cast: where the gamemaster's turn, the round it debriefs and the replay's cast are written, or empty for
// none; a round needs its turn. One the backend does not answer is no error: the replay is what the run needs, and they
// only add the round of the season scored in it, the countries called up and the players the clips name.
func Fetch(ctx context.Context, out, turn, round, cast string, config Config, logger *slog.Logger) error {
	config = config.withDefaults()

	source, err := ssh_replay_source.New(config.Replay.SSH, ssh_replay_source.OpenSSH)
	if err != nil {
		return fmt.Errorf("failed to build the source of the replay: %w", err)
	}

	fetch := log_fetch_replay.New(fetch_replay_usecase.New(
		replays.Request{Span: config.Replay.Span, Until: config.Replay.Until},
		cptime.SystemClock{}, source, file_replay_store.Store(out),
	), logger)

	replay, err := fetch.Execute(ctx)
	if err != nil {
		return err //nolint:wrapcheck // the use case already named what failed.
	}
	if cast != "" {
		// Its decorator warns that the cast is left out.
		_, _ = log_fetch_cast.New(fetch_cast_usecase.New(source, file_replay_store.CastStore(cast)), logger).
			Execute(ctx, replay.Window)
	}

	if turn == "" {
		return nil
	}
	// Their decorators warn that the turn or the round is left out.
	told, err := log_fetch_turn.New(fetch_turn_usecase.New(source, file_replay_store.TurnStore(turn)), logger).Execute(ctx)
	if err != nil || round == "" {
		return nil //nolint:nilerr // the decorator warned: a run with no turn still has its replay.
	}
	_, _ = log_fetch_round.New(fetch_round_usecase.New(source, file_replay_store.RoundStore(round)), logger).Execute(ctx, told)
	return nil
}

// What a run asks of the platforms: when each post goes out is the publishing module's calendar.
func planOf(config Config) posts.Plan {
	return posts.Plan{Review: config.Review, Platforms: config.Platforms, BestOnly: bestOnly}
}

// Each story to render, with the arm of each experiment that runs its clip is in (Choice.Assigned), and how the
// editor's call went.
func Choose(ctx context.Context, findings string, config Config, logger *slog.Logger) (Choice, error) {
	config = config.withDefaults()

	experiments, err := posts.ExperimentsOf(running...)
	if err != nil {
		return Choice{}, fmt.Errorf("the experiments that run: %w", err)
	}

	publishing, err := ssh_publishing.New(config.Replay.SSH, ssh_publishing.OpenSSH, &http.Client{Timeout: uploadTimeout})
	if err != nil {
		return Choice{}, fmt.Errorf("failed to reach the backend's publishing module: %w", err)
	}
	editor, err := editorOf(config.Anthropic)
	if err != nil {
		return Choice{}, err
	}

	choose := log_choose_clips.New(choose_clips_usecase.New(
		generator_output.FindingsFile(findings), log_history.New(publishing, logger), publishing, cptime.SystemClock{},
		choose_clips_usecase.Room{PerRun: config.PerRun, Plan: planOf(config)},
		experiments, posts.SystemDice{}, editor,
	), logger)

	return choose.Execute(ctx) //nolint:wrapcheck // the use case already named what failed.
}

// What makes a run's stories, and counts its prompt for `clips edit`.
type editor interface {
	choose_clips_usecase.Editor
	Count(ctx context.Context, offer posts.Offer) (int64, error)
}

// The editor on a client of Anthropic's API built here; with no key, one that sends nothing.
func editorOf(config cpanthropic.Config) (editor, error) {
	client, err := cpanthropic.NewClient(config)
	if errors.Is(err, cpanthropic.ErrNoKey) {
		return claude_editor.Unkeyed{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to build the client of Anthropic's API: %w", err)
	}
	return claude_editor.New(client, claude_editor.DefaultModel), nil
}

// What the editor would make of saved findings: the prompt it is sent, its size as the model counts it (0 with no
// key), and, unless only the prompt is asked for, the run it chose.
type Trial struct {
	Instructions string
	Prompt       string
	Tokens       int64
	Offer        posts.Offer
	Choice       Choice
}

// The editor tried by hand on saved findings, outside any run: every platform with room for PerRun clips, and what
// was told read from the backend's publishing module when an ssh host is set, else nothing told.
func Edit(ctx context.Context, findings string, promptOnly bool, config Config, logger *slog.Logger) (Trial, error) {
	config = config.withDefaults()
	now := cptime.SystemClock{}.Now()

	found, err := generator_output.FindingsFile(findings).Findings(ctx)
	if err != nil {
		return Trial{}, err //nolint:wrapcheck // the file already said what is wrong with it.
	}
	var told []posts.Telling
	if config.Replay.SSH != "" {
		publishing, err := ssh_publishing.New(config.Replay.SSH, ssh_publishing.OpenSSH, http.DefaultClient)
		if err != nil {
			return Trial{}, fmt.Errorf("failed to reach the backend's publishing module: %w", err)
		}
		if told, err = publishing.Told(ctx, posts.ToldSince(now)); err != nil {
			return Trial{}, fmt.Errorf("failed to read the stories already told: %w", err)
		}
	} else {
		logger.Warn("no CLICKPLANET_SSH: nothing counts as told")
	}

	editor, err := editorOf(config.Anthropic)
	if err != nil {
		return Trial{}, err
	}
	trial := Trial{Instructions: claude_editor.Instructions, Offer: found.Offer(config.PerRun, told, now)}
	if trial.Prompt, err = claude_editor.PromptOf(trial.Offer); err != nil {
		return Trial{}, err //nolint:wrapcheck // the editor already said what it could not write.
	}
	if trial.Tokens, err = editor.Count(ctx, trial.Offer); err != nil {
		logger.Warn("the prompt is not counted", slog.String("why", err.Error()))
	}
	if !promptOnly {
		trial.Choice = choose_clips_usecase.Edit(ctx, editor, found, config.PerRun, told, now)
	}
	return trial, nil
}

// clock: the time each post's review is counted from. Each post is planned and posted by the server's publishing module,
// over ssh. Then the server keeps who each post names: one it cannot keep is a warning, the post is out already.
func Schedule(ctx context.Context, folder string, config Config, clock cptime.Clock, logger *slog.Logger) (Report, error) {
	config = config.withDefaults()

	experiments, err := posts.ExperimentsOf(running...)
	if err != nil {
		return nil, fmt.Errorf("the experiments that run: %w", err)
	}

	publishing, err := ssh_publishing.New(config.Replay.SSH, ssh_publishing.OpenSSH, &http.Client{Timeout: uploadTimeout})
	if err != nil {
		return nil, fmt.Errorf("failed to reach the backend's publishing module: %w", err)
	}
	sender := posts.NewSender(publishing, log_service.New(publishing, logger), defaultBackoff, posts.SystemPause{})

	schedule := log_schedule_clips.New(schedule_clips_usecase.New(
		generator_output.Folder(folder), planOf(config), clock, posts.SystemDice{}, sender, experiments,
	), logger)

	report, err := schedule.Execute(ctx)
	if err != nil {
		return nil, err //nolint:wrapcheck // the use case already named what failed.
	}
	if len(report.Appearances()) == 0 {
		return report, nil
	}
	appearances, err := ssh_appearance_store.New(config.Replay.SSH, ssh_appearance_store.OpenSSH)
	if err != nil {
		return nil, fmt.Errorf("failed to build the store of the appearances: %w", err)
	}
	log_keep_appearances.New(keep_appearances_usecase.New(appearances), logger).Execute(ctx, report)
	return report, nil
}
