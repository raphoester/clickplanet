package clips

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/posts"
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
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type (
	Pick            = posts.Pick
	Platform        = posts.Platform
	Report          = posts.Report
	PlatformOutcome = posts.PlatformOutcome
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

	Zernio zernio.Config
}

type ReplayConfig struct {
	// user@host of the machine the backend runs on.
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
)

var (
	defaultBackoff   = posts.Backoff{time.Minute, 5 * time.Minute, 15 * time.Minute}
	defaultPlatforms = []Platform{posts.TikTok, posts.YouTube}
	// Posts in a trailing 24 hours. TikTok holds a post over 15 on an account of its Business app, and Zernio counts
	// those 15 over the same window: 14 leaves one for a post made by hand. YouTube lets a channel upload about 10 a day.
	dailyBudget = posts.Budget{posts.TikTok: 14, posts.YouTube: 8}
)

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

func Choose(ctx context.Context, shortlist string, config Config, logger *slog.Logger) ([]Pick, error) {
	config = config.withDefaults()

	history, err := zernio.New(config.Zernio, &http.Client{Timeout: historyTimeout})
	if err != nil {
		return nil, fmt.Errorf("failed to build the posting service: %w", err)
	}

	choose := log_choose_clips.New(choose_clips_usecase.New(
		generator_output.ShortlistFile(shortlist), log_history.New(history, logger), history, cptime.SystemClock{},
		choose_clips_usecase.Room{PerRun: config.PerRun, Budget: dailyBudget, Platforms: config.Platforms},
	), logger)

	choice, err := choose.Execute(ctx)
	if err != nil {
		return nil, err
	}
	return choice.Chosen.Picks(), nil
}

func Schedule(ctx context.Context, folder string, config Config, logger *slog.Logger) (Report, error) {
	config = config.withDefaults()
	clock := cptime.SystemClock{}

	service, err := zernio.New(config.Zernio, &http.Client{Timeout: serviceTimeout})
	if err != nil {
		return nil, fmt.Errorf("failed to build the posting service: %w", err)
	}
	sender := posts.NewSender(log_service.New(service, logger), defaultBackoff, posts.SystemPause{}, clock)

	schedule := log_schedule_clips.New(schedule_clips_usecase.New(
		generator_output.Folder(folder), service,
		posts.Plan{Review: config.Review, Spacing: config.Spacing, Platforms: config.Platforms, Budget: dailyBudget},
		clock, posts.SystemDice{}, sender,
	), logger)

	return schedule.Execute(ctx)
}
