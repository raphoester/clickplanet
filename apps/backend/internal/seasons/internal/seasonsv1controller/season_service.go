package seasonsv1controller

import (
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/seasons/v1/seasonsv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/seasonsv1controller/get_season_handler"
)

type SeasonService struct {
	get_season_handler.GetSeasonHandler
}

var _ seasonsv1connect.SeasonServiceHandler = SeasonService{}
