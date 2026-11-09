// The game's posts on its own TikTok and YouTube: when each goes out, within what each platform takes in a day, through
// the one posting service, Zernio. The jobs that post ask it over ssh, on the console listener.
package publishing

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"connectrpc.com/connect"

	"github.com/raphoester/clickplanet.lol-backend/generated/proto/publishing/v1/publishingv1connect"

	"github.com/raphoester/clickplanet.lol-backend/internal/publishing/internal/calendar"
	"github.com/raphoester/clickplanet.lol-backend/internal/publishing/internal/posts/usecases/create_upload_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/publishing/internal/posts/usecases/schedule_post_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/publishing/internal/posts/usecases/schedule_post_usecase/log_schedule_post"
	"github.com/raphoester/clickplanet.lol-backend/internal/publishing/internal/posts/zernio"
	"github.com/raphoester/clickplanet.lol-backend/internal/publishing/internal/publishingv1controller"
	"github.com/raphoester/clickplanet.lol-backend/internal/publishing/internal/publishingv1controller/create_media_upload_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/publishing/internal/publishingv1controller/get_room_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/publishing/internal/publishingv1controller/get_room_handler/room_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/publishing/internal/publishingv1controller/list_posts_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/publishing/internal/publishingv1controller/plan_post_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/publishing/internal/publishingv1controller/plan_post_handler/plan_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/publishing/internal/publishingv1controller/schedule_post_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpbootstrap"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

const moduleName = "publishing"

// Each call to the posting service: its calls carry a little JSON, never a video.
const serviceTimeout = time.Minute

func NewModule(config Config) cpbootstrap.Module {
	return cpbootstrap.Module{
		Name:    moduleName,
		Enabled: true,
		DiSequence: func(_ context.Context, props cpbootstrap.Props) error {
			return build(config, props)
		},
	}
}

func build(config Config, props cpbootstrap.Props) error {
	service, err := serviceOf(config, props)
	if err != nil {
		return err
	}
	if err := props.AdminRPC.Mount(func(options ...connect.HandlerOption) (string, http.Handler) {
		return publishingv1connect.NewAdminServiceHandler(service, options...)
	}); err != nil {
		return fmt.Errorf("failed to mount publishing.v1.AdminService: %w", err)
	}
	return nil
}

// Off with no key: every call says so, and the server still starts.
func serviceOf(config Config, props cpbootstrap.Props) (publishingv1connect.AdminServiceHandler, error) {
	if config.Zernio.APIKey == "" {
		props.Logger.Warn("publishing is off: no ZERNIO_API_KEY")
		return publishingv1controller.Off{}, nil
	}

	zernioService, err := zernio.New(zernio.Config{APIKey: config.Zernio.APIKey, BaseURL: config.Zernio.BaseURL},
		&http.Client{Timeout: serviceTimeout})
	if err != nil {
		return nil, fmt.Errorf("failed to build the posting service: %w", err)
	}
	clock := cptime.SystemClock{}

	props.Logger.Info("publishing built")
	return publishingv1controller.AdminService{
		CreateMediaUploadHandler: create_media_upload_handler.New(create_upload_usecase.New(zernioService)),
		GetRoomHandler:           get_room_handler.New(room_query.New(zernioService, calendar.Default, clock)),
		PlanPostHandler:          plan_post_handler.New(plan_query.New(zernioService, calendar.Default, clock)),
		SchedulePostHandler: schedule_post_handler.New(log_schedule_post.New(
			schedule_post_usecase.New(zernioService, calendar.Default, clock), props.Logger,
		)),
		ListPostsHandler: list_posts_handler.New(zernioService),
	}, nil
}

type Config struct {
	// The posting service's. No key, no posting: every call is FailedPrecondition.
	Zernio ZernioConfig
}

type ZernioConfig struct {
	APIKey string
	// Empty is Zernio's own API: a test's fake.
	BaseURL string
}

func (c Config) Validate() error {
	if c.Zernio.BaseURL == "" {
		return nil
	}
	if base, err := url.Parse(c.Zernio.BaseURL); err != nil || (base.Scheme != "https" && base.Scheme != "http") || base.Host == "" {
		return errors.New("publishing.zernio.baseURL is no http address")
	}
	return nil
}
