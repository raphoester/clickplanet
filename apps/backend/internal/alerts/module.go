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
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1/authv1connect"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1/planetv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/alertsv1controller"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/alertsv1controller/get_discord_link_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/alertsv1controller/get_discord_link_handler/link_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/alertsv1controller/get_player_link_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/alertsv1controller/get_player_link_handler/player_link_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/alertsv1controller/link_members_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/alertsv1controller/unlink_player_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/discord"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/discord/rest_discord"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/links/postgres_link_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/links/rpc_auth_discord_accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/links/rpc_auth_grants"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/links/rpc_planet_allegiance"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/links/usecases/follow_flag_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/links/usecases/forget_link_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/links/usecases/join_server_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/links/usecases/link_members_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/links/usecases/link_members_usecase/log_link_members"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/links/usecases/unlink_player_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/pings"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/pings/postgres_ping_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/pings/usecases/announce_start_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/pings/usecases/hear_take_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/roles"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/roles/postgres_role_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/roles/tidy_roles_runner"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/subscribers/allegiance_pledged_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/subscribers/battle_started_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/subscribers/discord_join_granted_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/subscribers/log_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/subscribers/player_forgotten_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/subscribers/season_started_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/subscribers/tile_taken_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpbootstrap"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
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
		return fmt.Errorf("the alerts module asks the planet module the flag a player pledged, and the auth module "+
			"what a Discord sign-in joins with and who signed in with Discord: %w", err)
	}
	planet := planetv1connect.NewInternalServiceClient(internal, baseURL)
	auth := authv1connect.NewInternalServiceClient(internal, baseURL)
	allegiances := rpc_planet_allegiance.New(planet)

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
			return cpredis.Subscribe(ctx, streams, "alerts-allegiances", log_subscriber.New(allegiance_pledged_subscriber.New(
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
			return cpredis.Subscribe(ctx, streams, "alerts-forgotten", log_subscriber.New(player_forgotten_subscriber.New(
				forget_link_usecase.New(linkStore)), props.Logger))
		},
		func() (cpbootstrap.Runner, error) {
			return cpredis.Subscribe(ctx, streams, "alerts-joins", log_subscriber.New(discord_join_granted_subscriber.New(
				rpc_auth_grants.New(auth),
				join_server_usecase.New(clock, linkStore, server, allegiances, book),
				props.Logger), props.Logger))
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
	props.Runners.Add(tidy_roles_runner.New(book, props.Logger))

	service := alertsv1controller.AlertsService{
		GetDiscordLinkHandler: get_discord_link_handler.New(link_query.NewPostgresQuery(db), config.Discord.serverURL()),
	}
	verifier := cpsessionverifier.New(props.Internal, props.Logger.With(slog.String("module", moduleName)))
	if err := props.RPC.Mount(func(options ...connect.HandlerOption) (string, http.Handler) {
		return alertsv1connect.NewAlertsServiceHandler(service, options...)
	},
		alertsv1controller.NewSessionInterceptor(verifier, clock, props.Metrics),
	); err != nil {
		return fmt.Errorf("failed to mount alerts.v1.AlertsService: %w", err)
	}

	adminService := alertsv1controller.AdminService{
		GetPlayerLinkHandler: get_player_link_handler.New(player_link_query.NewPostgresQuery(db)),
		UnlinkPlayerHandler:  unlink_player_handler.New(unlink_player_usecase.New(linkStore, book, server)),
		LinkMembersHandler: link_members_handler.New(log_link_members.New(link_members_usecase.New(
			clock, linkStore, rpc_auth_discord_accounts.New(auth), server, allegiances, book), props.Logger)),
	}
	if err := props.AdminRPC.Mount(func(options ...connect.HandlerOption) (string, http.Handler) {
		return alertsv1connect.NewAdminServiceHandler(adminService, options...)
	}); err != nil {
		return fmt.Errorf("failed to mount alerts.v1.AdminService: %w", err)
	}

	props.Logger.Info("alerts built",
		slog.String("schema", config.Database.Schema), slog.String("guild", config.Discord.GuildID))

	return nil
}

type Config struct {
	// Off, the service is not served: the game offers no link, a Discord sign-in joins no server, and nobody is
	// pinged. Keep auth.discord.joinServer off with it: nobody would hear the grants auth tells.
	Enabled bool

	Database cppg.Config
	Redis    cpredis.Config

	Discord DiscordConfig
	// The page each ping links to.
	GameURL string
}

// The bot of the game's Discord application, the one players sign in with, in the game's server.
type DiscordConfig struct {
	BotToken string

	GuildID          string
	AlertsChannelID  string
	BattlesChannelID string
}

func (c DiscordConfig) app() rest_discord.App {
	return rest_discord.App{BotToken: c.BotToken, Guild: discord.GuildID(c.GuildID)}
}

func (c DiscordConfig) channels() pings.Channels {
	return pings.Channels{Alerts: discord.ChannelID(c.AlertsChannelID), Battles: discord.ChannelID(c.BattlesChannelID)}
}

func (c DiscordConfig) serverURL() string {
	return "https://discord.com/channels/" + c.GuildID + "/" + c.AlertsChannelID
}

const defaultGameURL = "https://clickplanet.lol/play"

func (c Config) withDefaults() Config {
	if c.GameURL == "" {
		c.GameURL = defaultGameURL
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
	if c.Discord.BotToken == "" {
		errs = append(errs, errors.New("alerts.discord.botToken is empty"))
	}
	for name, id := range map[string]string{
		"guildId": c.Discord.GuildID, "alertsChannelId": c.Discord.AlertsChannelID, "battlesChannelId": c.Discord.BattlesChannelID,
	} {
		if err := discord.ValidID(id); err != nil {
			errs = append(errs, fmt.Errorf("alerts.discord.%s: %w", name, err))
		}
	}
	if page := c.withDefaults().GameURL; !isHTTPSPage(page) {
		errs = append(errs, fmt.Errorf("alerts.gameUrl is %q, and must be an https page", page))
	}
	return errors.Join(errs...)
}

func isHTTPSPage(page string) bool {
	parsed, err := url.Parse(page)
	return err == nil && parsed.Scheme == "https" && parsed.Host != ""
}
