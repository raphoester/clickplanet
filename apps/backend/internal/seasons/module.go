package seasons

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"connectrpc.com/connect"

	"github.com/raphoester/clickplanet.lol-backend/generated/proto/seasons/v1/seasonsv1connect"

	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/calendar"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/calendar/usecases/get_numbered_season_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/calendar/usecases/get_season_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/icalcontroller/finale_handler"
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
	seasons := calendar.New(config.Calendar)

	service := seasonsv1controller.SeasonService{
		GetSeasonHandler: get_season_handler.New(
			get_season_usecase.New(seasons, cptime.SystemClock{}),
		),
	}
	if err := props.RPC.Mount(func(options ...connect.HandlerOption) (string, http.Handler) {
		return seasonsv1connect.NewSeasonServiceHandler(service, options...)
	}, seasonsv1controller.NewCacheInterceptor()); err != nil {
		return fmt.Errorf("failed to mount seasons.v1.SeasonService: %w", err)
	}

	finale := finale_handler.New(
		get_numbered_season_usecase.New(seasons), cptime.SystemClock{}, props.Server.AllowedOrigin+"/play",
	)
	if err := props.HTTP.Handle(finale_handler.Pattern, finale); err != nil {
		return fmt.Errorf("failed to handle %s: %w", finale_handler.Pattern, err)
	}

	props.Logger.Info("seasons built", slog.Int("seasons", len(config.Calendar.List)))

	return nil
}

type Config struct {
	Calendar calendar.Config `koanf:",squash"`
}

func (c Config) Validate() error {
	if err := c.Calendar.Validate(); err != nil {
		return fmt.Errorf("seasons: %w", err)
	}
	return nil
}
