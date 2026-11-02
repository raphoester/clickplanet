package alerts

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"connectrpc.com/connect"

	"github.com/raphoester/clickplanet.lol-backend/generated/proto/alerts/v1/alertsv1connect"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1/planetv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/alertsv1controller"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/alertsv1controller/complete_discord_link_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/alertsv1controller/get_discord_link_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/alertsv1controller/get_discord_link_handler/link_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/alertsv1controller/start_discord_link_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/discord"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/discord/rest_discord"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/links/postgres_link_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/links/random_secret_generator"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/links/rpc_planet_allegiance"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/links/usecases/complete_link_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/links/usecases/follow_flag_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/links/usecases/forget_link_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/links/usecases/start_link_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/pings"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/pings/postgres_ping_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/pings/usecases/announce_start_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/pings/usecases/hear_take_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/roles"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/roles/postgres_role_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/subscribers/account_deleted_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/subscribers/battle_started_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/subscribers/flag_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/subscribers/log_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/subscribers/season_started_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/subscribers/tile_taken_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpbootstrap"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpratelimit"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpredis"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsessionverifier"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

const moduleName = "alerts"

const discordTimeout = 10 * time.Second

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
		return fmt.Errorf("the alerts module asks the planet module the flag a player pledged: %w", err)
	}
	planet := planetv1connect.NewInternalServiceClient(internal, baseURL)

	db := cppg.New(config.Database)
	if err := db.ConnectCtx(ctx); err != nil {
		return fmt.Errorf("failed to connect the alerts module to postgres: %w", err)
	}
	if err := db.Migrate(ctx, migrations.FS); err != nil {
		_ = db.Close()
		return fmt.Errorf("failed to migrate the %s schema: %w", config.Database.Schema, err)
	}

	streams := cpredis.New(config.Redis, props.Metrics)
	if err := streams.ConnectCtx(ctx); err != nil {
		_ = db.Close()
		return fmt.Errorf("failed to connect the alerts module to redis: %w", err)
	}
	closeStores := func() {
		_ = streams.Close()
		_ = db.Close()
	}

	server := rest_discord.New(config.Discord.app(), rest_discord.Production, &http.Client{Timeout: discordTimeout})
	linkStore := postgres_link_store.New(db)
	pingStore := postgres_ping_store.New(db)
	book := roles.NewBook(postgres_role_store.New(db), server)
	herald := pings.NewHerald(book, server, config.Discord.channels(), config.GameURL)
	announce := announce_start_usecase.New(clock, pingStore, herald, db)

	var subscriptions []cpbootstrap.Runner
	for _, subscribe := range []func() (cpbootstrap.Runner, error){
		func() (cpbootstrap.Runner, error) {
			return cpredis.Subscribe(ctx, streams, "alerts-attacks", log_subscriber.New(tile_taken_subscriber.New(
				hear_take_usecase.New(pings.NewWatch(), clock, pingStore, herald, db)), props.Logger))
		},
		func() (cpbootstrap.Runner, error) {
			return cpredis.Subscribe(ctx, streams, "alerts-flags", log_subscriber.New(flag_subscriber.New(
				follow_flag_usecase.New(linkStore, book, server)), props.Logger))
		},
		func() (cpbootstrap.Runner, error) {
			return cpredis.Subscribe(ctx, streams, "alerts-battles", log_subscriber.New(battle_started_subscriber.New(
				announce), props.Logger))
		},
		func() (cpbootstrap.Runner, error) {
			return cpredis.Subscribe(ctx, streams, "alerts-seasons", log_subscriber.New(season_started_subscriber.New(
				announce), props.Logger))
		},
		func() (cpbootstrap.Runner, error) {
			return cpredis.Subscribe(ctx, streams, "alerts-accounts", log_subscriber.New(account_deleted_subscriber.New(
				forget_link_usecase.New(linkStore)), props.Logger))
		},
	} {
		subscription, err := subscribe()
		if err != nil {
			closeStores()
			return fmt.Errorf("failed to subscribe the alerts module: %w", err)
		}
		subscriptions = append(subscriptions, subscription)
	}

	// Not a closer: closers run before the runners stop, and the runners use the pool.
	props.Runners.Add(cppg.CloseAfter(db, props.Logger, cpredis.CloseAfter(streams, props.Logger, subscriptions...)))

	limiter := cpratelimit.New("alerts-limiter", config.RateLimiter, clock)
	props.Runners.Add(limiter)

	serverURL := config.Discord.serverURL()
	service := alertsv1controller.AlertsService{
		GetDiscordLinkHandler: get_discord_link_handler.New(link_query.NewPostgresQuery(db), serverURL),
		StartDiscordLinkHandler: start_discord_link_handler.New(
			start_link_usecase.New(random_secret_generator.Generator{}, clock, linkStore, server)),
		CompleteDiscordLinkHandler: complete_discord_link_handler.New(
			complete_link_usecase.New(clock, linkStore, server, rpc_planet_allegiance.New(planet), book), serverURL),
	}
	verifier := cpsessionverifier.New(props.Internal, props.Logger.With(slog.String("module", moduleName)))
	if err := props.RPC.Mount(func(options ...connect.HandlerOption) (string, http.Handler) {
		return alertsv1connect.NewAlertsServiceHandler(service, options...)
	},
		alertsv1controller.NewSessionInterceptor(verifier, clock, props.Metrics),
		alertsv1controller.NewRateLimitInterceptor(limiter),
	); err != nil {
		return fmt.Errorf("failed to mount alerts.v1.AlertsService: %w", err)
	}

	props.Logger.Info("alerts built",
		slog.String("schema", config.Database.Schema), slog.String("guild", config.Discord.GuildID))

	return nil
}

type Config struct {
	// Off, the service is not served: the game offers no link and pings nobody.
	Enabled bool

	Database cppg.Config
	Redis    cpredis.Config

	Discord DiscordConfig
	// The page each ping links to.
	GameURL string

	RateLimiter cpratelimit.Config
}

// The game's Discord application, the one players sign in with, and its bot in the server.
type DiscordConfig struct {
	ClientID     string
	ClientSecret string
	BotToken     string
	// The sign-in's own: the page tells a link from a sign-in by what it started.
	RedirectURL string

	GuildID          string
	AlertsChannelID  string
	BattlesChannelID string
}

func (c DiscordConfig) app() rest_discord.App {
	return rest_discord.App{
		ClientID: c.ClientID, ClientSecret: c.ClientSecret, BotToken: c.BotToken, RedirectURL: c.RedirectURL,
		Guild: discord.GuildID(c.GuildID),
	}
}

func (c DiscordConfig) channels() pings.Channels {
	return pings.Channels{Alerts: discord.ChannelID(c.AlertsChannelID), Battles: discord.ChannelID(c.BattlesChannelID)}
}

func (c DiscordConfig) serverURL() string {
	return "https://discord.com/channels/" + c.GuildID + "/" + c.AlertsChannelID
}

const (
	defaultGameURL = "https://clickplanet.lol/play"
	// A start and a complete per link, and a try again.
	defaultBurst     = 6
	defaultPerSecond = 1.0 / 30
)

func (c Config) withDefaults() Config {
	if c.GameURL == "" {
		c.GameURL = defaultGameURL
	}
	if c.RateLimiter.PerSecond <= 0 {
		c.RateLimiter.PerSecond = defaultPerSecond
	}
	if c.RateLimiter.Burst <= 0 {
		c.RateLimiter.Burst = defaultBurst
	}
	return c
}

func (c Config) Validate() error {
	if !c.Enabled {
		return nil
	}
	var errs []error
	if err := c.Database.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("alerts.database: %w", err))
	}
	if err := c.Redis.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("alerts.redis: %w", err))
	}
	for name, value := range map[string]string{
		"clientId": c.Discord.ClientID, "clientSecret": c.Discord.ClientSecret, "botToken": c.Discord.BotToken,
	} {
		if value == "" {
			errs = append(errs, fmt.Errorf("alerts.discord.%s is empty", name))
		}
	}
	for name, id := range map[string]string{
		"guildId": c.Discord.GuildID, "alertsChannelId": c.Discord.AlertsChannelID, "battlesChannelId": c.Discord.BattlesChannelID,
	} {
		if err := discord.ValidID(id); err != nil {
			errs = append(errs, fmt.Errorf("alerts.discord.%s: %w", name, err))
		}
	}
	for name, page := range map[string]string{"discord.redirectUrl": c.Discord.RedirectURL, "gameUrl": c.withDefaults().GameURL} {
		if parsed, err := url.Parse(page); err != nil || parsed.Scheme != "https" || parsed.Host == "" {
			errs = append(errs, fmt.Errorf("alerts.%s is %q, and must be an https page", name, page))
		}
	}
	return errors.Join(errs...)
}
