package clipsv1controller

import (
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/clips/v1/clipsv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/clipsv1controller/get_appearances_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/clipsv1controller/set_name_shown_handler"
)

type ClipsService struct {
	get_appearances_handler.GetAppearancesHandler
	set_name_shown_handler.SetNameShownHandler
}

var _ clipsv1connect.ClipsServiceHandler = ClipsService{}
