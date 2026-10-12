package clips

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/posts"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/posts/generator_output"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/posts/log_service"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/posts/markdown_summary"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/posts/usecases/choose_clips_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/posts/usecases/choose_clips_usecase/log_choose_clips"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/posts/usecases/schedule_clips_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/posts/usecases/schedule_clips_usecase/log_schedule_clips"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/posts/zernio"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Config struct {
	// How many clips a run schedules, the best stories first.
	PerRun int
	// How long before the first clip of a run goes out: the time left to cancel it.
	Review time.Duration
	// Between two clips of one run.
	Spacing time.Duration

	Zernio zernio.Config
}

const (
	defaultPerRun  = 2
	defaultReview  = 4 * time.Hour
	defaultSpacing = 2 * time.Hour

	// A 22s clip is about 80 MB to upload.
	serviceTimeout = 10 * time.Minute
)

var defaultBackoff = posts.Backoff{time.Minute, 5 * time.Minute, 15 * time.Minute}

func (c Config) withDefaults() Config {
	if c.PerRun <= 0 {
		c.PerRun = defaultPerRun
	}
	if c.Review <= 0 {
		c.Review = defaultReview
	}
	if c.Spacing <= 0 {
		c.Spacing = defaultSpacing
	}
	return c
}

// Writes the picks of the stories to render as a JSON array, best first: none when no story is worth a clip.
func Choose(ctx context.Context, shortlist string, config Config, picks io.Writer, logger *slog.Logger) error {
	config = config.withDefaults()

	choose := log_choose_clips.New(choose_clips_usecase.New(generator_output.ShortlistFile(shortlist), config.PerRun), logger)

	chosen, err := choose.Execute(ctx)
	if err != nil {
		return err
	}
	if err := json.NewEncoder(picks).Encode(chosen.Picks()); err != nil {
		return fmt.Errorf("failed to write the picks: %w", err)
	}
	return nil
}

// Schedules every clip rendered in the folder, and writes what became of each one in the summary.
func Schedule(ctx context.Context, folder string, config Config, summary io.Writer, logger *slog.Logger) error {
	config = config.withDefaults()
	clock := cptime.SystemClock{}

	service, err := zernio.New(config.Zernio, &http.Client{Timeout: serviceTimeout})
	if err != nil {
		return fmt.Errorf("failed to build the posting service: %w", err)
	}
	sender := posts.NewSender(log_service.New(service, logger), defaultBackoff, posts.SystemPause{}, clock)

	schedule := log_schedule_clips.New(schedule_clips_usecase.New(
		generator_output.Folder(folder),
		posts.Cadence{Review: config.Review, Spacing: config.Spacing},
		clock, sender, markdown_summary.New(summary),
	), logger)

	_, err = schedule.Execute(ctx)
	return err
}
