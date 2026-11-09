package promo

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/teaser_job"
	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/teaser_job/ssh_backend"
	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/teaser_job/teaser_folder"
	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/teaser_job/usecases/list_due_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/teaser_job/usecases/list_due_usecase/log_list_due"
	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/teaser_job/usecases/post_teasers_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/teaser_job/usecases/post_teasers_usecase/log_post_teasers"
)

// The teasers job's, a binary of its own (cmd/teasers) that shares no process and no config with the server. It decides
// nothing: the module above does, and the job asks it over ssh, on the console listener.
type JobConfig struct {
	// user@host of the machine the backend runs on.
	SSH string
}

type (
	Due           = list_due_usecase.Kept
	TeasersReport = teaser_job.Report
)

const (
	TeaserPosted  = teaser_job.Posted
	TeaserHeld    = teaser_job.Held
	TeaserSkipped = teaser_job.Skipped
)

// A slideshow of a few cards is a few MB, straight to the posting service's storage.
const teaserUploadTimeout = 5 * time.Minute

var teaserBackoff = teaser_job.Backoff{time.Minute, 5 * time.Minute, 15 * time.Minute}

// Each battle promo says is due, its teaser.json written in a folder of its own under folder, for the renderer.
func ListDue(ctx context.Context, folder string, config JobConfig, logger *slog.Logger) ([]Due, error) {
	backend, err := ssh_backend.New(config.SSH, ssh_backend.OpenSSH, http.DefaultClient)
	if err != nil {
		return nil, fmt.Errorf("failed to reach the backend: %w", err)
	}
	return log_list_due.New(list_due_usecase.New(backend, teaser_folder.Folder(folder)), logger).Execute(ctx) //nolint:wrapcheck // the use case already named what failed.
}

// Each teaser rendered under folder uploaded and handed to promo, which posts it or says why not.
func PostTeasers(ctx context.Context, folder string, config JobConfig, logger *slog.Logger) (TeasersReport, error) {
	backend, err := ssh_backend.New(config.SSH, ssh_backend.OpenSSH, &http.Client{Timeout: teaserUploadTimeout})
	if err != nil {
		return nil, fmt.Errorf("failed to reach the backend: %w", err)
	}
	return log_post_teasers.New(post_teasers_usecase.New( //nolint:wrapcheck // the use case already named what failed.
		teaser_folder.Folder(folder), backend, backend, teaserBackoff, teaser_job.SystemPause{},
	), logger).Execute(ctx)
}
