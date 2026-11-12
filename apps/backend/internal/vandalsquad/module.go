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
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpredis"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
	"github.com/raphoester/clickplanet.lol-backend/internal/vandalsquad/internal/graffiti"
	"github.com/raphoester/clickplanet.lol-backend/internal/vandalsquad/internal/graffiti/log_graffiti"
	"github.com/raphoester/clickplanet.lol-backend/internal/vandalsquad/internal/graffiti/postgres_graffiti_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/vandalsquad/internal/graffiti/rpc_planet"
	"github.com/raphoester/clickplanet.lol-backend/internal/vandalsquad/internal/graffiti/usecases/anonymize_graffiti_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/vandalsquad/internal/graffiti/usecases/patrol_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/vandalsquad/internal/graffiti/uuid_id_provider"
	"github.com/raphoester/clickplanet.lol-backend/internal/vandalsquad/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/vandalsquad/internal/subscribers/log_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/vandalsquad/internal/subscribers/player_forgotten_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/vandalsquad/internal/vandalsquadv1controller"
	"github.com/raphoester/clickplanet.lol-backend/internal/vandalsquad/internal/vandalsquadv1controller/list_graffiti_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/vandalsquad/internal/vandalsquadv1controller/list_graffiti_handler/graffiti_query"
)

const moduleName = "vandalsquad"

// The squad that watches what players paint on the map: every minute it reads what each player painted in the last
// hour in planet's ledger, and flags each drawing that reads as a swastika. It reverts and bans nobody.
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
		return fmt.Errorf("the vandal squad asks planet what players painted and what the tiles hold now: %w", err)
	}
	planet := rpc_planet.New(planetv1connect.NewInternalServiceClient(internal, baseURL))

	db := cppg.New(config.Database)
	if err := db.ConnectCtx(ctx); err != nil {
		return fmt.Errorf("failed to connect the vandal squad to postgres: %w", err)
	}
	if err := db.Migrate(ctx, migrations.FS); err != nil {
		_ = db.Close()
		return fmt.Errorf("failed to migrate the %s schema: %w", config.Database.Schema, err)
	}

	streams := cpredis.New(config.Redis, props.Metrics)
	if err := streams.ConnectCtx(ctx); err != nil {
		_ = db.Close()
		return fmt.Errorf("failed to connect the vandal squad to redis: %w", err)
	}

	store := postgres_graffiti_store.New(db)

	forgotten, err := cpredis.Subscribe(ctx, streams, "vandalsquad-graffiti-forgotten",
		log_subscriber.New(player_forgotten_subscriber.New(anonymize_graffiti_usecase.New(store)), props.Logger))
	if err != nil {
		_ = streams.Close()
		_ = db.Close()
		return fmt.Errorf("failed to subscribe the graffiti to player.v1.PlayerForgotten: %w", err)
	}

	patrol := patrol_usecase.New(planet, store, log_graffiti.New(props.Logger), clock,
		graffiti.NewPatrol(config.Window, config.GameURL, uuid_id_provider.Provider{}))
	// Not a closer: closers run before the runners stop, and the runners use the pool.
	props.Runners.Add(cppg.CloseAfter(db, props.Logger,
		patrol_usecase.NewRunner(config.Interval, patrol, props.Logger),
		cpredis.CloseAfter(streams, props.Logger, forgotten)))

	admin := vandalsquadv1controller.AdminService{
		ListGraffitiHandler: list_graffiti_handler.New(graffiti_query.NewPostgresQuery(db)),
	}
	if err := props.AdminRPC.Mount(func(options ...connect.HandlerOption) (string, http.Handler) {
		return vandalsquadv1connect.NewAdminServiceHandler(admin, options...)
	}); err != nil {
		return fmt.Errorf("failed to mount vandalsquad.v1.AdminService: %w", err)
	}

	props.Logger.Info("vandal squad built", slog.String("schema", config.Database.Schema),
		slog.Duration("window", config.Window), slog.Duration("interval", config.Interval))
	return nil
}

type Config struct {
	// Off, it is never built: nothing reads planet's ledger, nothing is flagged, and vandalsquad.v1.AdminService 404s.
	Enabled bool

	// The flagged graffiti.
	Database cppg.Config
	// Where it hears that a player was forgotten.
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
	// Planet's ledger reads no more at once.
	longestWindow = 72 * time.Hour
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
	if err := c.Database.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("vandalSquad.database: %w", err))
	}
	if err := c.Redis.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("vandalSquad.redis: %w", err))
	}
	config := c.withDefaults()
	if page, err := url.Parse(config.GameURL); err != nil || page.Scheme != "https" || page.Host == "" {
		errs = append(errs, fmt.Errorf("vandalSquad.gameUrl is %q, and must be an https page", config.GameURL))
	}
	if config.Window > longestWindow {
		errs = append(errs, fmt.Errorf("vandalSquad.window is %s, longer than the %s planet's ledger reads at once",
			config.Window, longestWindow))
	}
	if config.Interval > config.Window {
		errs = append(errs, fmt.Errorf("vandalSquad.interval is %s, longer than its window of %s: a take would be "+
			"left out of every patrol", config.Interval, config.Window))
	}
	return errors.Join(errs...)
}
