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
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/posts/usecases/choose_clips_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/posts/usecases/choose_clips_usecase/log_choose_clips"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/posts/usecases/schedule_clips_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/posts/usecases/schedule_clips_usecase/log_schedule_clips"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/posts/zernio"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/replays"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/replays/file_replay_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/replays/ssh_replay_source"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/replays/usecases/fetch_race_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/replays/usecases/fetch_race_usecase/log_fetch_race"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/replays/usecases/fetch_replay_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/replays/usecases/fetch_replay_usecase/log_fetch_replay"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/replays/usecases/fetch_season_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/replays/usecases/fetch_season_usecase/log_fetch_season"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/stats"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/stats/ssh_stats_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/stats/usecases/take_snapshot_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/stats/usecases/take_snapshot_usecase/log_take_snapshot"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpanthropic"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type (
	Pick            = posts.Pick
	Assignment      = posts.Assignment
	Choice          = posts.Choice
	Editorial       = posts.Editorial
	Variants        = posts.Variants
	Platform        = posts.Platform
	Report          = posts.Report
	Outcome         = posts.Outcome
	Sound           = posts.Sound
	Track           = posts.Track
	PlatformOutcome = posts.PlatformOutcome
	StatsReport     = stats.Report
)

type Config struct {
	Replay ReplayConfig

	PerRun int
	// Zero is none: a post goes out as soon as its clip is rendered.
	Review time.Duration
	// The least time between two posts on one platform.
	Spacing time.Duration
	// Each clip is one post on all of them that have room left in their day.
	Platforms []Platform
	// The country of TikTok's trending chart a clip of the experiment "sound" drawn for a trending track gets one from:
	// ISO 3166-1 alpha-2. Empty is defaultSoundCountry.
	SoundCountry string
	// The posts a snapshot of their numbers reads: those scheduled since then. Zero is the last 4 weeks.
	StatsSince time.Time

	Zernio zernio.Config
	// The editor's. No key is no editor: every run drawn for the model falls back to the rules, and says why.
	Anthropic cpanthropic.Config
}

type ReplayConfig struct {
	// user@host of the machine the backend runs on: where the replay is asked, and the snapshots of stats kept.
	SSH  string
	Span time.Duration
	// Zero is now.
	Until time.Time
}

const (
	// The ledger keeps this much in memory: a replay cannot reach further back.
	defaultSpan = 72 * time.Hour

	// A run every 2 hours: 2 a run is 24 a day at most, which the budget below brings down.
	defaultPerRun  = 2
	defaultSpacing = time.Hour

	// A 22s clip is about 80 MB to upload.
	serviceTimeout = 10 * time.Minute
	historyTimeout = time.Minute
	chartTimeout   = time.Minute
	statsTimeout   = time.Minute
)

var (
	defaultBackoff   = posts.Backoff{time.Minute, 5 * time.Minute, 15 * time.Minute}
	defaultPlatforms = []Platform{posts.TikTok, posts.YouTube}
	// Posts in a trailing 24 hours. TikTok holds a post over 15 on an account of its Business app, and Zernio counts
	// those 15 over the same window: 14 leaves one for a post made by hand. YouTube lets a channel upload about 10 a day.
	dailyBudget = posts.Budget{posts.TikTok: 14, posts.YouTube: 8}
)

const TikTok = posts.TikTok

var ErrChartCountry = posts.ErrChartCountry

// Where most of the clips' TikTok views are expected to come from until the stats say: the largest audience of a
// game captioned in English. The arm records the chart it read, so a change shows in the analysis.
const defaultSoundCountry = "US"

func ChartCountryOf(code string) (string, error) {
	return posts.ChartCountryOf(code) //nolint:wrapcheck // a pure delegation: the domain already named what is wrong.
}

// What a call of input tokens in and output out costs the editor's model.
func SpendOf(input, output int64) posts.Spend {
	return claude_editor.SpendOf(claude_editor.DefaultModel, input, 0, 0, output)
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
	if c.Spacing <= 0 {
		c.Spacing = defaultSpacing
	}
	if len(c.Platforms) == 0 {
		c.Platforms = defaultPlatforms
	}
	if c.SoundCountry == "" {
		c.SoundCountry = defaultSoundCountry
	}
	return c
}

// race, season: where each is written, or empty for none. One the backend does not answer is no error: the replay is
// what the run needs, and they only add the day of the season closed in it and the countries called up.
func Fetch(ctx context.Context, out, race, season string, config Config, logger *slog.Logger) error {
	config = config.withDefaults()

	source, err := ssh_replay_source.New(config.Replay.SSH, ssh_replay_source.OpenSSH)
	if err != nil {
		return fmt.Errorf("failed to build the source of the replay: %w", err)
	}

	fetch := log_fetch_replay.New(fetch_replay_usecase.New(
		replays.Request{Span: config.Replay.Span, Until: config.Replay.Until},
		cptime.SystemClock{}, source, file_replay_store.Store(out),
	), logger)

	if _, err := fetch.Execute(ctx); err != nil {
		return err
	}

	// Their decorators warn that the race or the season is left out.
	if race != "" {
		_, _ = log_fetch_race.New(fetch_race_usecase.New(source, file_replay_store.RaceStore(race)), logger).Execute(ctx)
	}
	if season != "" {
		_, _ = log_fetch_season.New(fetch_season_usecase.New(source, file_replay_store.SeasonStore(season)), logger).Execute(ctx)
	}
	return nil
}

// Each story to render, with the arm of each experiment that runs its clip is in (Choice.Assigned), and who chose them.
func Choose(ctx context.Context, shortlist string, config Config, logger *slog.Logger) (Choice, error) {
	config = config.withDefaults()

	experiments, err := posts.ExperimentsOf(running...)
	if err != nil {
		return Choice{}, fmt.Errorf("the experiments that run: %w", err)
	}

	history, err := zernio.New(config.Zernio, &http.Client{Timeout: historyTimeout})
	if err != nil {
		return Choice{}, fmt.Errorf("failed to build the posting service: %w", err)
	}
	editor, err := editorOf(config.Anthropic)
	if err != nil {
		return Choice{}, err
	}

	choose := log_choose_clips.New(choose_clips_usecase.New(
		generator_output.ShortlistFile(shortlist), log_history.New(history, logger), history, cptime.SystemClock{},
		choose_clips_usecase.Room{PerRun: config.PerRun, Budget: dailyBudget, Platforms: config.Platforms},
		experiments, posts.SystemDice{}, editor,
	), logger)

	return choose.Execute(ctx) //nolint:wrapcheck // the use case already named what failed.
}

// What chooses a run drawn for the model, and counts its prompt for `clips edit`.
type editor interface {
	choose_clips_usecase.Editor
	Count(ctx context.Context, offer posts.Offer) (int64, error)
}

// The editor on a client of Anthropic's API built here; with no key, one that sends nothing and always falls back.
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

// What the editor would make of a saved shortlist: the prompt it is sent, its size as the model counts it (0 with
// no key), and, unless only the prompt is asked for, the run it chose.
type Trial struct {
	Instructions string
	Prompt       string
	Tokens       int64
	Offer        posts.Offer
	Choice       Choice
	Headlines    map[Pick]string
}

// The editor tried by hand on a saved shortlist, outside any run: every platform with room for PerRun clips, and what
// was told read from the posting service when a Zernio key is set, else nothing told.
func Edit(ctx context.Context, shortlist string, promptOnly bool, config Config, logger *slog.Logger) (Trial, error) {
	config = config.withDefaults()
	now := cptime.SystemClock{}.Now()

	stories, err := generator_output.ShortlistFile(shortlist).Shortlist(ctx)
	if err != nil {
		return Trial{}, err //nolint:wrapcheck // the file already said what is wrong with it.
	}
	var told []posts.Telling
	if config.Zernio.APIKey != "" {
		history, err := zernio.New(config.Zernio, &http.Client{Timeout: historyTimeout})
		if err != nil {
			return Trial{}, fmt.Errorf("failed to build the posting service: %w", err)
		}
		if told, err = history.Told(ctx, posts.ToldSince(now)); err != nil {
			return Trial{}, fmt.Errorf("failed to read the stories already told: %w", err)
		}
	} else {
		logger.Warn("no ZERNIO_API_KEY: nothing counts as told")
	}

	editor, err := editorOf(config.Anthropic)
	if err != nil {
		return Trial{}, err
	}
	trial := Trial{Instructions: claude_editor.Instructions, Offer: stories.Offer(config.PerRun, told, now)}
	if trial.Prompt, err = claude_editor.PromptOf(trial.Offer); err != nil {
		return Trial{}, err //nolint:wrapcheck // the editor already said what it could not write.
	}
	if trial.Tokens, err = editor.Count(ctx, trial.Offer); err != nil {
		logger.Warn("the prompt is not counted", slog.String("why", err.Error()))
	}
	if !promptOnly {
		trial.Choice, trial.Headlines = choose_clips_usecase.Edit(ctx, editor, stories, config.PerRun, told, now)
	}
	return trial, nil
}

func Schedule(ctx context.Context, folder string, config Config, logger *slog.Logger) (Report, error) {
	config = config.withDefaults()
	clock := cptime.SystemClock{}

	experiments, err := posts.ExperimentsOf(running...)
	if err != nil {
		return nil, fmt.Errorf("the experiments that run: %w", err)
	}

	service, err := zernio.New(config.Zernio, &http.Client{Timeout: serviceTimeout})
	if err != nil {
		return nil, fmt.Errorf("failed to build the posting service: %w", err)
	}
	sender := posts.NewSender(log_service.New(service, logger), defaultBackoff, posts.SystemPause{}, clock)
	// A chart that hangs falls back in a minute, not in the upload's 10.
	charts, err := zernio.New(config.Zernio, &http.Client{Timeout: chartTimeout})
	if err != nil {
		return nil, fmt.Errorf("failed to build the posting service: %w", err)
	}

	schedule := log_schedule_clips.New(schedule_clips_usecase.New(
		generator_output.Folder(folder), service,
		posts.Plan{Review: config.Review, Spacing: config.Spacing, Platforms: config.Platforms, Budget: dailyBudget},
		clock, posts.SystemDice{}, sender, experiments, charts, posts.Soundtracks{Country: config.SoundCountry},
	), logger)

	return schedule.Execute(ctx)
}

// Keeps a snapshot of the numbers of every post of the job in the backend's postgres, over ssh, with what each post
// was. A post that could not be read is in the report, not the error: the error is for a snapshot that could not be
// taken or kept at all.
func Stats(ctx context.Context, config Config, logger *slog.Logger) (StatsReport, error) {
	source, err := zernio.New(config.Zernio, &http.Client{Timeout: statsTimeout})
	if err != nil {
		return StatsReport{}, fmt.Errorf("failed to build the posting service: %w", err)
	}
	store, err := ssh_stats_store.New(config.Replay.SSH, ssh_stats_store.OpenSSH)
	if err != nil {
		return StatsReport{}, fmt.Errorf("failed to build the store of the snapshots: %w", err)
	}

	snapshot := log_take_snapshot.New(take_snapshot_usecase.New(
		source, store, cptime.SystemClock{}, config.StatsSince, defaultBackoff, posts.SystemPause{},
	), logger)

	return snapshot.Execute(ctx)
}
