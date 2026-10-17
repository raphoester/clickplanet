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
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/replays/usecases/fetch_replay_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/replays/usecases/fetch_replay_usecase/log_fetch_replay"
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

	PerRun  int
	Review  time.Duration
	Spacing time.Duration
	// Each clip is one post on all of them.
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

	defaultPerRun  = 6
	defaultReview  = 4 * time.Hour
	defaultSpacing = 2 * time.Hour

	// A 22s clip is about 80 MB to upload.
	serviceTimeout = 10 * time.Minute
	historyTimeout = time.Minute
)

var (
	defaultBackoff   = posts.Backoff{time.Minute, 5 * time.Minute, 15 * time.Minute}
	defaultPlatforms = []Platform{posts.TikTok, posts.YouTube}
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
	if c.Review <= 0 {
		c.Review = defaultReview
	}
	if c.Spacing <= 0 {
		c.Spacing = defaultSpacing
	}
	if len(c.Platforms) == 0 {
		c.Platforms = defaultPlatforms
	}
	return c
}

func Fetch(ctx context.Context, out string, config Config, logger *slog.Logger) error {
	config = config.withDefaults()

	source, err := ssh_replay_source.New(config.Replay.SSH, ssh_replay_source.OpenSSH)
	if err != nil {
		return fmt.Errorf("failed to build the source of the replay: %w", err)
	}

	fetch := log_fetch_replay.New(fetch_replay_usecase.New(
		replays.Request{Span: config.Replay.Span, Until: config.Replay.Until},
		cptime.SystemClock{}, source, file_replay_store.Store(out),
	), logger)

	_, err = fetch.Execute(ctx)
	return err
}

func Choose(ctx context.Context, shortlist string, config Config, logger *slog.Logger) ([]Pick, error) {
	config = config.withDefaults()

	history, err := zernio.New(config.Zernio, &http.Client{Timeout: historyTimeout})
	if err != nil {
		return nil, fmt.Errorf("failed to build the posting service: %w", err)
	}

	choose := log_choose_clips.New(choose_clips_usecase.New(
		generator_output.ShortlistFile(shortlist), log_history.New(history, logger), cptime.SystemClock{}, config.PerRun,
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
		generator_output.Folder(folder),
		posts.Plan{Review: config.Review, Spacing: config.Spacing, Platforms: config.Platforms},
		clock, posts.SystemDice{}, sender,
	), logger)

	return schedule.Execute(ctx)
}
