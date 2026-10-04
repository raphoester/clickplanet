package seasons

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"connectrpc.com/connect"

	"github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1/planetv1connect"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/seasons/v1/seasonsv1connect"

	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/calendar"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/calendar/usecases/get_season_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/finale"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/finale/rpc_planet_rules"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/finale/usecases/converge_rules_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/finale/usecases/converge_rules_usecase/log_converge_rules"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/lead"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/lead/rpc_planet_shares"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/lead/usecases/watch_lead_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/lead/usecases/watch_lead_usecase/log_watch_lead"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/seasonsv1controller"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/seasonsv1controller/get_season_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpbootstrap"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

const moduleName = "seasons"

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
	service := seasonsv1controller.SeasonService{
		GetSeasonHandler: get_season_handler.New(
			get_season_usecase.New(calendar.New(config.Calendar), cptime.SystemClock{}),
		),
	}
	if err := props.RPC.Mount(func(options ...connect.HandlerOption) (string, http.Handler) {
		return seasonsv1connect.NewSeasonServiceHandler(service, options...)
	}, seasonsv1controller.NewCacheInterceptor()); err != nil {
		return fmt.Errorf("failed to mount seasons.v1.SeasonService: %w", err)
	}

	internal, baseURL, err := props.Internal.Dial()
	if err != nil {
		return fmt.Errorf("the seasons module sets planet's rules: %w", err)
	}
	planet := planetv1connect.NewInternalServiceClient(internal, baseURL)

	seasons := calendar.New(config.Calendar)
	clock := cptime.SystemClock{}

	rules := log_converge_rules.New(converge_rules_usecase.New(
		seasons, finale.NewRules(config.Finale), clock, rpc_planet_rules.New(planet)), props.Logger)
	watch := log_watch_lead.New(watch_lead_usecase.New(
		seasons, config.Lead, clock, rpc_planet_shares.New(planet), props.Events), props.Logger)
	props.Runners.Add(converge_rules_usecase.NewRunner(config.Finale.WithDefaults().CheckEvery, rules, watch))

	props.Logger.Info("seasons built", slog.Int("seasons", len(config.Calendar.List)))

	return nil
}

type Config struct {
	Calendar calendar.Config `koanf:",squash"`

	Finale finale.Config
	Lead   lead.Config
}

func (c Config) Validate() error {
	if err := c.Calendar.Validate(); err != nil {
		return fmt.Errorf("seasons: %w", err)
	}
	if err := c.Finale.Validate(); err != nil {
		return fmt.Errorf("seasons: %w", err)
	}
	return nil
}
