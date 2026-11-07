package vandalsquad

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"connectrpc.com/connect"

	"github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1/planetv1connect"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/vandalsquad/v1/vandalsquadv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpbootstrap"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpredis"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
	"github.com/raphoester/clickplanet.lol-backend/internal/vandalsquad/internal/graffiti"
	"github.com/raphoester/clickplanet.lol-backend/internal/vandalsquad/internal/graffiti/log_graffiti"
	"github.com/raphoester/clickplanet.lol-backend/internal/vandalsquad/internal/graffiti/rpc_planet_map"
	"github.com/raphoester/clickplanet.lol-backend/internal/vandalsquad/internal/graffiti/usecases/patrol_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/vandalsquad/internal/paint"
	"github.com/raphoester/clickplanet.lol-backend/internal/vandalsquad/internal/subscribers/log_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/vandalsquad/internal/subscribers/tile_taken_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/vandalsquad/internal/vandalsquadv1controller"
	"github.com/raphoester/clickplanet.lol-backend/internal/vandalsquad/internal/vandalsquadv1controller/list_graffiti_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/vandalsquad/internal/vandalsquadv1controller/list_graffiti_handler/graffiti_query"
)

const moduleName = "vandalsquad"

// The graffiti kept for the operator's list, newest first.
const bookCapacity = 500

// The squad that watches what players paint on the map: it hears planet's takes, looks every minute at what each
// player painted in the last hour, and flags each drawing that reads as a swastika. It reverts and bans nobody.
func NewModule(config Config) cpbootstrap.Module {
	return cpbootstrap.Module{
		Name:    moduleName,
		Enabled: config.Enabled,
		DiSequence: func(ctx context.Context, props cpbootstrap.Props) error {
			return build(ctx, config.withDefaults(), props)
		},
	}
}

func build(ctx context.Context, config Config, props cpbootstrap.Props) error {
	clock := cptime.SystemClock{}

	internal, baseURL, err := props.Internal.Dial()
	if err != nil {
		return fmt.Errorf("the vandal squad asks planet what the tiles players painted hold now: %w", err)
	}
	planet := rpc_planet_map.New(planetv1connect.NewInternalServiceClient(internal, baseURL))

	streams := cpredis.New(config.Redis, props.Metrics)
	if err := streams.ConnectCtx(ctx); err != nil {
		return fmt.Errorf("failed to connect the vandal squad to redis: %w", err)
	}

	window := paint.NewWindow(config.Window, clock)
	book := graffiti.NewBook(bookCapacity)

	takes, err := cpredis.Subscribe(ctx, streams, "vandalsquad-takes",
		log_subscriber.New(tile_taken_subscriber.New(window), props.Logger))
	if err != nil {
		_ = streams.Close()
		return fmt.Errorf("failed to subscribe the vandal squad to planet's takes: %w", err)
	}
	props.Runners.Add(cpredis.CloseAfter(streams, props.Logger, takes))
	props.Runners.Add(patrol_usecase.NewRunner(config.Interval,
		patrol_usecase.New(window, planet, book, log_graffiti.New(props.Logger), clock, config.GameURL), props.Logger))

	admin := vandalsquadv1controller.AdminService{
		ListGraffitiHandler: list_graffiti_handler.New(graffiti_query.NewMemoryQuery(book)),
	}
	if err := props.AdminRPC.Mount(func(options ...connect.HandlerOption) (string, http.Handler) {
		return vandalsquadv1connect.NewAdminServiceHandler(admin, options...)
	}); err != nil {
		return fmt.Errorf("failed to mount vandalsquad.v1.AdminService: %w", err)
	}

	props.Logger.Info("vandal squad built",
		slog.Duration("window", config.Window), slog.Duration("interval", config.Interval))
	return nil
}

type Config struct {
	// Off, it is never built: nothing hears planet's takes, nothing is flagged, and vandalsquad.v1.AdminService 404s.
	Enabled bool

	Redis cpredis.Config
	// The page each graffiti's link opens.
	GameURL string
	// How far back a patrol looks at what each player painted: a drawing painted over longer is seen in part.
	Window time.Duration
	// How often it looks: a drawing is flagged this long after it is whole, at most.
	Interval time.Duration
}

const (
	defaultGameURL  = "https://clickplanet.lol/play"
	defaultWindow   = time.Hour
	defaultInterval = time.Minute
)

func (c Config) withDefaults() Config {
	if c.GameURL == "" {
		c.GameURL = defaultGameURL
	}
	if c.Window <= 0 {
		c.Window = defaultWindow
	}
	if c.Interval <= 0 {
		c.Interval = defaultInterval
	}
	return c
}

func (c Config) Validate() error {
	if !c.Enabled {
		return nil
	}
	var errs []error
	if err := c.Redis.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("vandalSquad.redis: %w", err))
	}
	config := c.withDefaults()
	if page, err := url.Parse(config.GameURL); err != nil || page.Scheme != "https" || page.Host == "" {
		errs = append(errs, fmt.Errorf("vandalSquad.gameUrl is %q, and must be an https page", config.GameURL))
	}
	if config.Interval > config.Window {
		errs = append(errs, fmt.Errorf("vandalSquad.interval is %s, longer than its window of %s: a take would be "+
			"forgotten before a patrol saw it", config.Interval, config.Window))
	}
	return errors.Join(errs...)
}
