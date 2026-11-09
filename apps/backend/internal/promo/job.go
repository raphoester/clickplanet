package promo

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/teasers"
	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/teasers/ssh_backend"
	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/teasers/teaser_folder"
	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/teasers/usecases/list_due_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/teasers/usecases/list_due_usecase/log_list_due"
	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/teasers/usecases/post_teasers_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/teasers/usecases/post_teasers_usecase/log_post_teasers"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

// The teasers job's, a binary of its own (cmd/teasers) that shares no process and no config with the server: what it
// asks of the server (promo's battles to come, the publishing module's posts) goes over ssh to the console listener.
type JobConfig struct {
	// user@host of the machine the backend runs on.
	SSH string
}

type (
	Due            = list_due_usecase.Due
	TeasersReport  = teasers.Report
	TeaserPlatform = teasers.Platform
)

const (
	TeaserScheduled = teasers.Scheduled
	TeaserHeld      = teasers.Held
	TeaserSkipped   = teasers.Skipped
)

// A slideshow of a few cards is a few MB, straight to the posting service's storage.
const teaserUploadTimeout = 5 * time.Minute

var teaserBackoff = teasers.Backoff{time.Minute, 5 * time.Minute, 15 * time.Minute}

// Each battle within the next 36 hours that no post teases yet, its teaser.json written in a folder of its own under
// folder, for the renderer.
func ListDue(ctx context.Context, folder string, config JobConfig, logger *slog.Logger) ([]Due, error) {
	backend, err := ssh_backend.New(config.SSH, ssh_backend.OpenSSH, http.DefaultClient)
	if err != nil {
		return nil, fmt.Errorf("failed to reach the backend: %w", err)
	}
	return log_list_due.New(list_due_usecase.New( //nolint:wrapcheck // the use case already named what failed.
		backend, backend, teaser_folder.Folder(folder), cptime.SystemClock{},
	), logger).Execute(ctx)
}

// Each teaser rendered under folder whose battle is still due, posted through the server's publishing module at the
// first slot its calendar gives by the battle: on TikTok and YouTube both if it can, else on one of them.
func PostTeasers(ctx context.Context, folder string, config JobConfig, logger *slog.Logger) (TeasersReport, error) {
	backend, err := ssh_backend.New(config.SSH, ssh_backend.OpenSSH, &http.Client{Timeout: teaserUploadTimeout})
	if err != nil {
		return nil, fmt.Errorf("failed to reach the backend: %w", err)
	}
	return log_post_teasers.New(post_teasers_usecase.New( //nolint:wrapcheck // the use case already named what failed.
		backend, backend, teaser_folder.Folder(folder),
		teasers.NewSender(backend, teaserBackoff, teasers.SystemPause{}), cptime.SystemClock{},
	), logger).Execute(ctx)
}
